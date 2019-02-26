package types

import (
	"errors"
	"fmt"
	"github.com/sony/sonyflake"
	"log"
	"strconv"
)

type UidGenerator struct {
	generator *sonyflake.Sonyflake
}

func (ug *UidGenerator) Init() error {
	ug.generator = sonyflake.NewSonyflake(sonyflake.Settings{})
	if ug.generator == nil {
		return errors.New("Initialize UID generator failed.")
	}
	return nil
}

func (ug *UidGenerator) newUid() (string, error) {
	id, err := ug.generator.NextID()
	if err != nil {
		log.Fatalf("flake.NextID() failed:   %s\n", err)
	}
	return strconv.FormatUint(id, 10), nil
}

func (ug *UidGenerator) NewSessionUid() (string, error) {
	id, err := ug.newUid()
	if err != nil {
		log.Fatalf("UidGenerator.NewSessionUid() failed:   %s\n", err)
	}
	idStr := fmt.Sprintf("session-%s", id)
	return idStr, nil
}

func (ug *UidGenerator) NewMsgUid() (string, error) {
	id, err := ug.newUid()
	if err != nil {
		log.Fatalf("UidGenerator.NewMsgUid() failed:   %s\n", err)
	}
	idStr := fmt.Sprintf("msg-%s", id)
	return idStr, nil
}
