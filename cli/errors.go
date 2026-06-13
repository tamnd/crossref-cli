package cli

import (
	"errors"

	"github.com/tamnd/crossref-cli/crossref"
)

func isNotFound(err error) bool {
	return errors.Is(err, crossref.ErrNotFound)
}
