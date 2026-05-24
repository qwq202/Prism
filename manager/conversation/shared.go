package conversation

import (
	"chat/auth"
	"chat/globals"
	"chat/utils"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"
)

type SharedPreviewForm struct {
	Name           string    `json:"name"`
	ConversationId int64     `json:"conversation_id"`
	Time           time.Time `json:"time"`
	Hash           string    `json:"hash"`
}

type SharedForm struct {
	Username string            `json:"username"`
	Name     string            `json:"name"`
	Messages []globals.Message `json:"messages"`
	Model    string            `json:"model"`
	Time     time.Time         `json:"time"`
}

type SharedHashForm struct {
	Id             int64 `json:"id"`
	ConversationId int64 `json:"conversation_id"`
	Refs           []int `json:"refs"`
}

func GetRef(refs []int) (result string) {
	for _, v := range refs {
		result += strconv.Itoa(v) + ","
	}
	return strings.TrimSuffix(result, ",")
}

func normalizeShareRefs(conversation *Conversation, refs []int) ([]int, error) {
	if conversation == nil || conversation.GetMessageLength() == 0 {
		return nil, errors.New("conversation has no messages")
	}

	if len(refs) == 0 {
		return []int{-1}, nil
	}

	normalized := make([]int, 0, len(refs))
	seen := make(map[int]bool)
	for _, ref := range refs {
		if ref == -1 {
			return []int{-1}, nil
		}
		if ref < 0 || seen[ref] || !conversation.HasMessageId(ref) {
			continue
		}
		normalized = append(normalized, ref)
		seen[ref] = true
	}

	if len(normalized) == 0 {
		return nil, errors.New("no valid messages to share")
	}

	return normalized, nil
}

func sharedConversationModel(shared *SharedForm) string {
	if shared == nil || strings.TrimSpace(shared.Model) == "" {
		return globals.GPT3Turbo
	}
	return shared.Model
}

func ShareConversation(db *sql.DB, user *auth.User, id int64, refs []int) (string, error) {
	if id <= 0 || user == nil {
		return "", errors.New("conversation not found")
	}

	userID := user.GetID(db)
	if userID <= 0 {
		return "", errors.New("user not found")
	}

	conversation := LoadConversation(db, userID, id)
	if conversation == nil {
		return "", errors.New("conversation not found")
	}

	normalizedRefs, err := normalizeShareRefs(conversation, refs)
	if err != nil {
		return "", err
	}

	ref := GetRef(normalizedRefs)
	hash := utils.Md5EncryptForm(SharedHashForm{
		Id:             userID,
		ConversationId: id,
		Refs:           normalizedRefs,
	})

	if _, err := globals.ExecDb(db, `
			INSERT INTO sharing (hash, user_id, conversation_id, refs) VALUES (?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE refs = ?, updated_at = CURRENT_TIMESTAMP
		`, hash, userID, id, ref, ref); err != nil {
		return "", err
	}

	return hash, nil
}

func GetSharedMessages(db *sql.DB, userId int64, conversationId int64, refs []string) []globals.Message {
	conversation := LoadConversation(db, userId, conversationId)
	if conversation == nil {
		return nil
	}

	messages := make([]globals.Message, 0)
	for _, v := range refs {
		if v == "-1" {
			return conversation.GetMessage()
		} else {
			id, err := strconv.Atoi(v)
			if err != nil {
				continue
			}
			if !conversation.HasMessageId(id) {
				continue
			}
			messages = append(messages, conversation.GetMessageById(id))
		}
	}
	return messages
}

func ListSharedConversation(db *sql.DB, user *auth.User) []SharedPreviewForm {
	if user == nil {
		return nil
	}

	id := user.GetID(db)
	rows, err := globals.QueryDb(db, `
		SELECT conversation.conversation_name, conversation.conversation_id, sharing.updated_at, sharing.hash
		FROM sharing
		INNER JOIN conversation 
		    ON conversation.conversation_id = sharing.conversation_id 
		    AND conversation.user_id = sharing.user_id
		WHERE sharing.user_id = ?
		ORDER BY sharing.updated_at DESC
		LIMIT 100
	`, id)
	if err != nil {
		return nil
	}

	result := make([]SharedPreviewForm, 0)
	for rows.Next() {
		var updated []uint8
		var form SharedPreviewForm
		if err := rows.Scan(&form.Name, &form.ConversationId, &updated, &form.Hash); err != nil {
			continue
		}

		form.Time = *utils.ConvertTime(updated)
		result = append(result, form)
	}
	return result
}

func DeleteSharedConversation(db *sql.DB, user *auth.User, hash string) error {
	if user == nil {
		return nil
	}

	id := user.GetID(db)
	if _, err := globals.ExecDb(db, `
		DELETE FROM sharing WHERE user_id = ? AND hash = ?
	`, id, hash); err != nil {
		return err
	}
	return nil
}

func GetSharedConversation(db *sql.DB, hash string) (*SharedForm, error) {
	var shared SharedForm
	var (
		uid     int64
		cid     int64
		ref     string
		updated []uint8
	)
	if err := globals.QueryRowDb(db, `
		SELECT auth.username, sharing.refs, sharing.updated_at, conversation.conversation_name,
		       sharing.user_id, sharing.conversation_id, conversation.model
		FROM sharing
		INNER JOIN auth ON auth.id = sharing.user_id
		INNER JOIN conversation ON conversation.conversation_id = sharing.conversation_id AND conversation.user_id = sharing.user_id
		WHERE sharing.hash = ?
	`, hash).Scan(&shared.Username, &ref, &updated, &shared.Name, &uid, &cid, &shared.Model); err != nil {
		return nil, err
	}

	shared.Time = *utils.ConvertTime(updated)
	refs := strings.Split(ref, ",")
	shared.Messages = GetSharedMessages(db, uid, cid, refs)

	return &shared, nil
}

func UseSharedConversation(db *sql.DB, user *auth.User, hash string) *Conversation {
	shared, err := GetSharedConversation(db, hash)
	if err != nil {
		return nil
	}

	if user == nil {
		// anonymous
		return &Conversation{
			Auth:    false,
			UserID:  -1,
			Id:      -1,
			Name:    shared.Name,
			Message: shared.Messages,
			Model:   sharedConversationModel(shared),
		}
	}

	// create new conversation
	id := user.GetID(db)
	return &Conversation{
		Auth:      true,
		Id:        GetConversationLengthByUserID(db, id) + 1,
		UserID:    id,
		Name:      shared.Name,
		Model:     sharedConversationModel(shared),
		Message:   shared.Messages,
		Persisted: false,
	}
}

func (c *Conversation) LoadSharing(db *sql.DB, hash string) {
	if strings.TrimSpace(hash) == "" || c.Shared == true {
		return
	}

	shared, err := GetSharedConversation(db, hash)
	if err != nil {
		return
	}

	c.InsertMessages(shared.Messages, 0)
	c.SetName(db, shared.Name)
	c.Shared = true
}
