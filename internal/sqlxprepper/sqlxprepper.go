package sqlxprepper

import (
	"errors"

	"github.com/jmoiron/sqlx"
)

// Prepper is the struct that holds the error
type Prepper struct {
	panic  bool
	handle *sqlx.DB
	err    error
}

// NewPrepper returns a new Prepper
func NewPrepper(db *sqlx.DB) *Prepper {
	return &Prepper{
		panic:  true,
		handle: db,
	}
}

// DoNotPanic allows execution to continue after an sql error
// Errors must be checked separtely
func (p *Prepper) DoNotPanic() {
	p.panic = false
}

// Prep runs Preparex and stashed the error
func (p *Prepper) Prep(query string) *sqlx.Stmt {
	if p.err != nil {
		return nil
	}

	stmt, err := p.handle.Preparex(query)
	if err != nil {
		p.err = errors.New("Error preparing statmement " + query + " " + err.Error())
		if p.panic {
			panic(p.err)
		}

		return nil
	}

	return stmt
}

// GetErr returns the error if any
func (p *Prepper) GetErr() error {
	return p.err
}

// PanicIfError panics if an error is set
func (p *Prepper) PanicIfError() {
	if p.err != nil {
		panic(p.err)
	}
}
