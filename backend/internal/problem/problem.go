package problem

type Error struct {
	Status  int
	Message string
	Data    any
}

func (e *Error) Error() string             { return e.Message }
func New(status int, message string) error { return &Error{Status: status, Message: message} }
func Validation(field, message string) error {
	if field == "" {
		field = "_form"
	}
	return &Error{Status: 400, Message: "提交的数据无效", Data: map[string]any{"code": "VALIDATION_ERROR", "fieldErrors": map[string]string{field: message}}}
}
