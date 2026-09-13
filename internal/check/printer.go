package check

import (
	"errors"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func englishPrinter() *message.Printer { return message.NewPrinter(language.English) }

func errorsAs(err error, target any) bool { return errors.As(err, target) }
