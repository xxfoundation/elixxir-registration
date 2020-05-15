////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package node

const (
	// Used for an operational node.
	// Default status of any node
	Online = iota
	// Used for an offline node. Defined as a node that hasn't
	//   polled permissioning for an extended period of time
	// An offline node can be reverted to an online node if resumes polling
	Offline
	// An OutOfNetwork node is defined as a node which is
	//  no longer considered for teaming.  Any node outOfNetwork has it's
	//  round cancelled and is not considered for future teams.
	// fixme: if/when reinstatement is implemented, document here
	OutOfNetwork
)
