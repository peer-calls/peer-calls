package server

import "fmt"

var NotImplementedErr = fmt.Errorf("Not implemented")

func firstError(errors ...error) error {
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}
