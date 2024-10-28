package main

type UserExistsError struct {
	errMsg string
}

func (e *UserExistsError) Error() string {
  return e.errMsg
}
