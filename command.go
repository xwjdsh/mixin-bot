package bot

import (
	"encoding/base64"

	"github.com/fox-one/mixin-sdk-go"
	"github.com/gofrs/uuid"
)

type Command interface {
	Name() string
	Execute(s *Session, msg *mixin.MessageView) (*mixin.MessageRequest, error)
}

type generalCommand struct {
	name string
	step int
}

func (c *generalCommand) Name() string {
	return c.name
}

func (c *generalCommand) Step() int {
	return c.step
}

type echoCommand struct {
	generalCommand
}

func (c *echoCommand) Execute(s *Session, msg *mixin.MessageView) (*mixin.MessageRequest, error) {
	if s.CurrentStep == 0 && msg.Data == "" {
		s.CurrentStep += 1
		return nil, nil
	}

	id, _ := uuid.FromString(msg.MessageID)
	data := base64.StdEncoding.EncodeToString([]byte(msg.Data))
	reply := &mixin.MessageRequest{
		ConversationID: msg.ConversationID,
		RecipientID:    msg.UserID,
		MessageID:      uuid.NewV5(id, "reply").String(),
		Category:       msg.Category,
		Data:           data,
	}

	s.Command = ""
	return reply, nil
}
