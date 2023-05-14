package x

type Server struct{}

func (Server) Method(s string) int {
	return len(s)
}
