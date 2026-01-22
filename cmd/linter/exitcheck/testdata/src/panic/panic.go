// Package panic содержит тестовые кейсы для проверки детекции panic.
package panic

func testPanic() {
	panic("this is a panic") // want "usage of builtin panic is discouraged"
}

func testPanicWithRecover() {
	defer func() {
		recover()
	}()
	panic("recovered") // want "usage of builtin panic is discouraged"
}

// CustomPanic - пользовательская функция с именем panic
func CustomPanic(msg string) {
	// Это не builtin panic, поэтому не должно вызывать ошибку
}

func testCustomPanic() {
	// Эта функция - не builtin, но для простоты тестов
	// мы тут просто показываем что другие функции не детектятся
	msg := "test"
	_ = msg
}
