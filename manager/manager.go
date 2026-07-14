package manager

import (
	autotitle "chat/addition/title"
	"chat/auth"
	"chat/globals"
	"chat/manager/conversation"
	"chat/utils"
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const chatRequestAckCapability = "chat_request_ack_v1"

type WebsocketAuthForm struct {
	Token string `json:"token" binding:"required"`
	Id    int64  `json:"id" binding:"required"`
	Ref   string `json:"ref"`
}

func ParseAuth(c *gin.Context, token string) *auth.User {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}

	if strings.HasPrefix(token, "Bearer ") {
		token = token[7:]
	}

	return auth.ParseToken(c, token)
}

func splitMessage(message string) (int, string, error) {
	parts := strings.SplitN(message, ":", 2)
	if len(parts) == 2 {
		if id, err := strconv.Atoi(parts[0]); err == nil {
			return id, parts[1], nil
		}
	}

	return 0, message, fmt.Errorf("message type error")
}

func getId(message string) (int, error) {
	if id, err := strconv.Atoi(message); err == nil {
		return id, nil
	}

	return 0, fmt.Errorf("message type error")
}

func maybeAutoTitle(conn *Connection, user *auth.User, instance *conversation.Conversation) {
	if user == nil || instance == nil {
		return
	}

	if instance.CountMessagesByRole(globals.User) != 1 || instance.CountMessagesByRole(globals.Assistant) != 1 {
		return
	}

	title := autotitle.GenerateConversationTitle(
		auth.GetGroup(conn.GetDB(), user),
		instance.GetMessage(),
		conn.GetCache(),
	)
	if strings.TrimSpace(title) == "" {
		return
	}

	name := utils.Extract(title, 50, "...")
	instance.Name = name
	if !instance.RenameConversation(conn.GetDB(), name) {
		return
	}
	_ = conn.SendClient(globals.ChatSegmentResponse{
		Title: name,
	})
}

func refreshPersistedConversation(db *sql.DB, userID int64, instance *conversation.Conversation) *conversation.Conversation {
	if db == nil || instance == nil || instance.IsTransient() || !instance.Persisted || instance.GetId() == -1 {
		return instance
	}

	if fresh := conversation.LoadConversation(db, userID, instance.GetId()); fresh != nil {
		return fresh
	}

	return instance
}

func hasVisibleAssistantText(message globals.Message) bool {
	return strings.TrimSpace(message.Content) != ""
}

func sendChatRequestState(conn *Connection, requestID string, conversationID int64, status string, accepted bool, retryable bool, message string, end bool) {
	if strings.TrimSpace(requestID) == "" {
		return
	}
	_ = conn.SendClient(globals.ChatSegmentResponse{
		Conversation:  conversationID,
		RequestID:     requestID,
		RequestStatus: status,
		Accepted:      accepted,
		Retryable:     retryable,
		Message:       message,
		End:           end,
	})
}

type chatRequestLeaseGuard struct {
	db         *sql.DB
	userID     int64
	requestID  string
	ownerToken string
	stop       chan struct{}
	done       chan struct{}
	lost       atomic.Bool
}

func startChatRequestLeaseGuard(db *sql.DB, userID int64, requestID string, ownerToken string) (*chatRequestLeaseGuard, bool) {
	guard := &chatRequestLeaseGuard{
		db: db, userID: userID, requestID: requestID, ownerToken: ownerToken,
		stop: make(chan struct{}), done: make(chan struct{}),
	}
	if !guard.renew() {
		return guard, false
	}
	go func() {
		defer close(guard.done)
		ticker := time.NewTicker(conversation.ChatRequestLeaseHeartbeatInterval())
		defer ticker.Stop()
		for {
			select {
			case <-guard.stop:
				return
			case <-ticker.C:
				if !guard.renew() {
					guard.lost.Store(true)
					return
				}
			}
		}
	}()
	return guard, true
}

func (g *chatRequestLeaseGuard) renew() bool {
	owned, err := conversation.RenewChatRequestLease(g.db, g.userID, g.requestID, g.ownerToken)
	return err == nil && owned
}

func (g *chatRequestLeaseGuard) stillOwns() bool {
	return !g.lost.Load() && g.renew()
}

func (g *chatRequestLeaseGuard) close() {
	close(g.stop)
	<-g.done
}

func ChatAPI(c *gin.Context) {
	var conn *utils.WebSocket
	if conn = utils.NewWebsocket(c, false); conn == nil {
		return
	}
	defer conn.DeferClose()

	db := utils.GetDBFromContext(c)

	authForm, err := utils.ReadForm[WebsocketAuthForm](conn)
	if err != nil {
		return
	}

	user := ParseAuth(c, authForm.Token)
	authenticated := user != nil

	id := auth.GetId(db, user)

	instance := conversation.ExtractConversation(db, user, authForm.Id, authForm.Ref)
	hash := fmt.Sprintf(":chatthread:%s", utils.Md5Encrypt(utils.Multi(
		authenticated,
		strconv.FormatInt(id, 10),
		c.ClientIP(),
	)))

	buf := NewConnection(conn, authenticated, hash, 10)
	buf.Handle(func(form *conversation.FormMessage) error {
		switch form.Type {
		case CapabilitiesType:
			_ = buf.SendClient(globals.ChatSegmentResponse{
				ResponseType: "capabilities",
				Capabilities: []string{chatRequestAckCapability},
			})
		case ChatType:
			requestID := strings.TrimSpace(form.RequestID)
			if form.Transient {
				instance.SetTransient(true)
				if err := instance.AddMessageFromForm(form); err != nil {
					return err
				}
				sendChatRequestState(buf, requestID, instance.GetId(), conversation.ChatRequestAccepted, true, false, "", false)
				response := ChatHandler(buf, user, instance, false)
				instance.SaveResponse(nil, response)
				return nil
			}

			instance = refreshPersistedConversation(db, id, instance)
			record, owner, reserveErr := conversation.ReserveChatRequest(db, id, requestID, instance.GetId())
			if reserveErr != nil {
				sendChatRequestState(buf, requestID, instance.GetId(), conversation.ChatRequestReserved, false, true, "", false)
				break
			}
			if !owner {
				accepted := record != nil && record.Status != conversation.ChatRequestReserved
				conversationID := instance.GetId()
				status := conversation.ChatRequestReserved
				if record != nil {
					conversationID = record.ConversationID
					status = record.Status
				}
				sendChatRequestState(buf, requestID, conversationID, status, accepted, !accepted, "", false)
				break
			}
			messagePersisted := false
			if hasPendingAskUserCall(instance) {
				if record != nil {
					_ = conversation.ReleaseChatRequestOwned(db, id, requestID, record.OwnerToken)
				}
				sendChatRequestState(buf, requestID, instance.GetId(), "rejected", false, false, "Answer or skip the pending question before sending a new message.", true)
				if requestID == "" {
					_ = buf.SendClient(globals.ChatSegmentResponse{
						Message: "Answer or skip the pending question before sending a new message.",
						End:     true,
					})
				}
				break
			} else if record != nil && record.Recovered {
				instance = conversation.LoadConversation(db, id, record.ConversationID)
				messagePersisted = instance != nil && instance.HasRequestID(requestID)
			} else if requestID != "" && record != nil && !instance.Persisted {
				messagePersisted = instance.HandleNewMessageWithRequest(db, form, record.OwnerToken)
			} else {
				messagePersisted = instance.HandleMessage(db, form)
			}
			if messagePersisted {
				if requestID != "" {
					owned := false
					var err error
					if record != nil {
						owned, err = conversation.MarkChatRequestStatusOwned(
							db, id, requestID, instance.GetId(), record.OwnerToken, conversation.ChatRequestAccepted,
						)
					}
					if err != nil {
						owned = false
					}
					if !owned {
						sendChatRequestState(buf, requestID, instance.GetId(), conversation.ChatRequestReserved, false, true, "", false)
						break
					}
					sendChatRequestState(buf, requestID, instance.GetId(), conversation.ChatRequestAccepted, true, false, "", false)
				}

				var lease *chatRequestLeaseGuard
				if requestID != "" && record != nil {
					var leaseOwned bool
					lease, leaseOwned = startChatRequestLeaseGuard(db, id, requestID, record.OwnerToken)
					if !leaseOwned {
						sendChatRequestState(buf, requestID, instance.GetId(), conversation.ChatRequestReserved, false, true, "", false)
						break
					}
				}
				response := ChatHandler(buf, user, instance, false)
				response.RequestID = requestID
				leaseOwned := lease == nil || lease.stillOwns()
				responseSaved := leaseOwned && instance.SaveResponse(db, response)
				if responseSaved {
					if hasVisibleAssistantText(response) {
						maybeAutoTitle(buf, user, instance)
					}
				}
				if requestID != "" && responseSaved {
					completed, err := conversation.MarkChatRequestStatusOwned(
						db, id, requestID, instance.GetId(), record.OwnerToken, conversation.ChatRequestCompleted,
					)
					if err == nil && completed {
						sendChatRequestState(buf, requestID, instance.GetId(), conversation.ChatRequestCompleted, true, false, "", false)
					}
				}
				if lease != nil {
					lease.close()
				}
			} else if requestID != "" {
				if record != nil {
					_ = conversation.ReleaseChatRequestOwned(db, id, requestID, record.OwnerToken)
				}
				sendChatRequestState(buf, requestID, instance.GetId(), conversation.ChatRequestReserved, false, true, "", false)
			}
		case StopType:
			break
		case ShareType:
			instance.LoadSharing(db, form.Message)
		case RestartType:
			instance = refreshPersistedConversation(db, id, instance)
			// reset the params if set
			instance.ApplyParam(form)

			response := ChatHandler(buf, user, instance, true)
			if instance.SaveResponse(db, response) {
				if hasVisibleAssistantText(response) {
					maybeAutoTitle(buf, user, instance)
				}
			}
		case MaskType:
			instance.LoadMask(form.Message)
		case EditType:
			if id, message, err := splitMessage(form.Message); err == nil {
				instance.EditMessage(id, message)
				instance.SaveConversation(db)
			} else {
				return err
			}
		case RemoveType:
			id, err := getId(form.Message)
			if err != nil {
				return err
			}

			instance.RemoveMessage(id)
			instance.SaveConversation(db)
		case ToolResultType:
			instance = refreshPersistedConversation(db, id, instance)
			toolMessage, answerErr := buildAskUserAnswerMessage(instance, form)
			if answerErr != nil {
				_ = buf.SendClient(globals.ChatSegmentResponse{
					Message: answerErr.Error(),
					End:     true,
				})
				break
			}

			instance.ApplyParam(form)
			instance.AddMessage(toolMessage)
			if !instance.IsTransient() && !instance.SaveConversation(db) {
				_ = buf.SendClient(globals.ChatSegmentResponse{
					Message: "Failed to save the answer. Please try again.",
					End:     true,
				})
				break
			}

			response := ChatHandler(buf, user, instance, false)
			responseDB := db
			if instance.IsTransient() {
				responseDB = nil
			}
			if instance.SaveResponse(responseDB, response) {
				if hasVisibleAssistantText(response) {
					maybeAutoTitle(buf, user, instance)
				}
			}
		}

		return nil
	})
}
