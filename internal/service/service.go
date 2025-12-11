package service

type Service interface {
	Install() error
	Uninstall() error
	Status() (string, error)
}

func New() Service {
	return newPlatformService()
}
