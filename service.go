package zerosvc

// service definition
type Service struct {
	// path relative to node name
	Path        string      `json:"path"`
	Description interface{} `json:"description"`
	Defaults    interface{} `json:"defaults,omitempty"`
}

func NewService() Service {
	var s Service
	return s
}
