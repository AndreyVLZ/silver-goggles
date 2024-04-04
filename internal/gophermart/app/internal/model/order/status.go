package order

type Status uint8

const (
	StatusNotSupport Status = iota // вместо ошибки
	StatusNew                      // NEW								//		db
	StatusRegistered               // REGISTERED						//acc
	StatusProcessing               // PROCESSING						//acc	db
	StatusInvalid                  // INVALID		// окончательный	//acc	db
	StatusProcessed                // PROCESSED		// окончательный	//acc	db
	StatusWithdraw                 // withdrawal	// окончательный
)

func supportStatus() [7]string {
	return [7]string{
		"status NOT support",
		"NEW",
		"REGISTERED",
		"PROCESSING",
		"INVALID",
		"PROCESSED",
		"withdrawal",
	}
}

func ParseStatus(status string) Status {
	statuses := supportStatus()
	for i := range statuses {
		if status == statuses[i] {
			return Status(i)
		}
	}
	return StatusNew
}

func (s Status) String() string { return supportStatus()[s] }
