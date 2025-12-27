package state

type State struct{}

func Load() (*State, error) {
    return &State{}, nil
}
