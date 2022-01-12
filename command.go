package bot

import "github.com/fox-one/mixin-sdk-go"

type generalCommand struct {
	name string
	step int
}

func (c *generalCommand) Step() int {
	return c.step
}

func (c *generalCommand) Name() string {
	return c.name
}

type printCommand struct {
	generalCommand
}

func (c *printCommand) Execute(step int, message *mixin.MessageView) (*mixin.MessageRequest, error) {
	return nil, nil
}
