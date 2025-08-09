package ssh

import "fmt"

type FileInfoPayload struct {
	FileID  string
	EditUrl string
}

func (f FileInfoPayload) String() string {
	return fmt.Sprintf("FileID: %s, EditUrl: %s", f.FileID, f.EditUrl)
}
