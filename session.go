package bot

type UserSession struct {
	State       int // 0 -> general; 1 -> command
	currentStep int
}
