package bot

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/fox-one/mixin-sdk-go"
	"github.com/gofrs/uuid"
)

var commands = []Command{
	echoCommandInstance,
	helpCommandInstance,
}

type Command interface {
	Name() string
	Desc() string
	Execute(s *Session, msg *mixin.MessageView) (*mixin.MessageRequest, error)
}

type generalCommand struct {
	name string
	step int
	desc string
}

func (c *generalCommand) Name() string {
	return c.name
}

func (c *generalCommand) Desc() string {
	return c.desc
}

var echoCommandInstance = &echoCommand{
	generalCommand{
		name: "/echo",
		step: 1,
		desc: "Echo message, it can be in the form of '/echo content', or send '/echo' first and then content.",
	}}

type echoCommand struct {
	generalCommand
}

func (c *echoCommand) Execute(s *Session, msg *mixin.MessageView) (*mixin.MessageRequest, error) {
	data := msg.Data
	empty := s.CurrentStep == 0 && msg.Data == ""
	if empty {
		data = "Waiting to receive echo content..."
		s.CurrentStep += 1
	}

	id, _ := uuid.FromString(msg.MessageID)
	reply := &mixin.MessageRequest{
		ConversationID: msg.ConversationID,
		RecipientID:    msg.UserID,
		MessageID:      uuid.NewV5(id, "reply").String(),
		Category:       msg.Category,
		Data:           base64.StdEncoding.EncodeToString([]byte(data)),
	}

	if !empty {
		s.Command = ""
	}
	return reply, nil
}

var helpCommandInstance = &helpCommand{
	generalCommand{
		name: "/help",
		desc: "List all available commands.",
	}}

type helpCommand struct {
	generalCommand
}

func (c *helpCommand) Execute(s *Session, msg *mixin.MessageView) (*mixin.MessageRequest, error) {
	id, _ := uuid.FromString(msg.MessageID)
	result := []string{}
	for _, command := range commands {
		result = append(result, fmt.Sprintf("%-10v%s", command.Name(), command.Desc()))
	}

	data := base64.StdEncoding.EncodeToString([]byte(strings.Join(result, "\n")))
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
