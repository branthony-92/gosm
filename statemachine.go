package sm

import (
	"fmt"
)

// state IDs
const (
	StateIDError = iota
	FirstStateID
)

// event IDs
const (
	EventIDError = iota
	FirstEventID
)

type StateID = int
type EventID = int

type State struct {
	funcTable       map[string]func()
	transitionTable map[EventID]StateID
}

func NewState() *State {
	s := State{
		transitionTable: make(map[EventID]StateID),
		funcTable:       make(map[string]func()),
	}
	return &s
}

func (s *State) SetEntryHandler(fn func()) {
	s.funcTable["onEntry"] = fn
}
func (s *State) SetTicHandler(fn func()) {
	s.funcTable["onTic"] = fn
}
func (s *State) SetExitHandler(fn func()) {
	s.funcTable["onExit"] = fn
}

func (s *State) AddTransition(event EventID, targetState StateID) {
	s.transitionTable[event] = targetState
}

func (s *State) nextState(id EventID) StateID {
	if newStateID, ok := s.transitionTable[id]; ok {
		return newStateID
	}
	return StateIDError
}

type EventListener struct {
	eventCh chan EventID // channel to send events to the state machine
	doneCh  chan struct{}
}

func NewEventListener(buffer int) *EventListener {
	e := EventListener{
		eventCh: make(chan EventID, buffer),
		doneCh:  make(chan struct{}, buffer),
	}
	return &e
}

func (e *EventListener) PostEvent(eventID EventID) {
	e.eventCh <- eventID
}
func (e *EventListener) Shutdown() {
	e.doneCh <- struct{}{}
}

type StateMachine struct {
	stateMap     map[StateID]*State
	currentState StateID
}

func (s *StateMachine) callStateHandler(tag string) bool {
	fn, ok := s.stateMap[s.currentState].funcTable[tag]
	if ok && fn != nil {
		fn()
		return true
	}
	return false
}
func (s *StateMachine) transition(eventID EventID) bool {
	// get the current state's exit function
	s.callStateHandler("onExit")

	// get the current state struct
	currState := s.GetState(s.currentState)
	if currState == nil {
		fmt.Printf("Current State %v is nil\n", s.currentState)
		return false
	}

	// figure out our next state should be from the current state's
	// transition map and update the current state ID
	newStateID := currState.nextState(eventID)
	if newStateID == StateIDError {
		fmt.Println("Failed to retrieve next state")
		return false
	}
	s.currentState = newStateID

	// call the new state's onEnter handler
	s.callStateHandler("onEnter")
	return true
}

// Exported SM Methods
func NewStateMachine() *StateMachine {
	sm := StateMachine{
		stateMap:     make(map[StateID]*State),
		currentState: StateIDError,
	}
	return &sm
}

func (s *StateMachine) AddState(id StateID, state *State) {
	s.stateMap[id] = state
}

func (s *StateMachine) GetState(id StateID) *State {
	if state, ok := s.stateMap[id]; ok {
		return state
	}
	return nil
}

func (s *StateMachine) Initialze(initialState StateID) {
	s.currentState = initialState
	s.callStateHandler("onEnter")
}

func (s *StateMachine) Tic(e *EventListener) (bool, error) {
	if e == nil {
		return false, fmt.Errorf("No event listener passed in")
	}
	select {
	case eventID := <-e.eventCh:
		fmt.Printf("New Event: %v\n", eventID)
		if ok := s.transition(eventID); !ok {
			return false, fmt.Errorf("State transition failed")
		}
	case <-e.doneCh:
		fmt.Println("Done")
		return false, nil
	default:
		if !s.callStateHandler("onTic") {
			return false, fmt.Errorf("Could Not call OnTic")
		}
	}
	return true, nil
}
