package domain

type InvalidOperationError struct {
	Message string
}

func (e InvalidOperationError) Error() string {
	return e.Message
}

func InvalidOperation(message string) error {
	return InvalidOperationError{Message: message}
}

type KeyNotFoundError struct {
	Message string
}

func (e KeyNotFoundError) Error() string {
	return e.Message
}

func KeyNotFound(message string) error {
	return KeyNotFoundError{Message: message}
}

type ArgumentError struct {
	Message string
}

func (e ArgumentError) Error() string {
	return e.Message
}

func Argument(message string) error {
	return ArgumentError{Message: message}
}
