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
)

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

func ChatAPI(c *gin.Context) {
	var conn *utils.WebSocket
	if conn = utils.NewWebsocket(c, false); conn == nil {
		return
	}
	defer conn.DeferClose()

	db := utils.GetDBFromContext(c)

	form, err := utils.ReadForm[WebsocketAuthForm](conn)
	if err != nil {
		return
	}

	user := ParseAuth(c, form.Token)
	authenticated := user != nil

	id := auth.GetId(db, user)

	instance := conversation.ExtractConversation(db, user, form.Id, form.Ref)
	hash := fmt.Sprintf(":chatthread:%s", utils.Md5Encrypt(utils.Multi(
		authenticated,
		strconv.FormatInt(id, 10),
		c.ClientIP(),
	)))

	buf := NewConnection(conn, authenticated, hash, 10)
	buf.Handle(func(form *conversation.FormMessage) error {
		switch form.Type {
		case ChatType:
			if form.Transient {
				instance.SetTransient(true)
				if err := instance.AddMessageFromForm(form); err != nil {
					return err
				}
				response := ChatHandler(buf, user, instance, false)
				instance.SaveResponse(nil, response)
				return nil
			}

			instance = refreshPersistedConversation(db, id, instance)
			if hasPendingAskUserCall(instance) {
				_ = buf.SendClient(globals.ChatSegmentResponse{
					Message: "Answer or skip the pending question before sending a new message.",
					End:     true,
				})
				break
			}
			if instance.HandleMessage(db, form) {
				response := ChatHandler(buf, user, instance, false)
				if instance.SaveResponse(db, response) {
					if hasVisibleAssistantText(response) {
						maybeAutoTitle(buf, user, instance)
					}
				}
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
