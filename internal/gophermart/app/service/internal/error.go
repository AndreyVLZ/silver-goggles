package internal

type ErrFieldConflict struct {
	Field  any
	ErrStr string
}

func (e ErrFieldConflict) Data() any     { return e.Field }
func (e ErrFieldConflict) Error() string { return e.ErrStr }
