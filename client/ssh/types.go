package ssh

import "fmt"

type FileInfoPayload struct {
	FileID  string
	EditUrl string
}

func (f FileInfoPayload) String() string {
	return fmt.Sprintf("Your edit url: %s , DO NOT SHARE TO STRANGERS!", f.EditUrl)
}
