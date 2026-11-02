// Package provensearch answers a conjunctive word search from an index tree together with
// the proof of the answer, and judges such an answer from the proof alone. One rule serves
// both sides: the answering node stops when the rule calls its own proof complete, and the
// verifier applies the same rule to the proof it receives. An answer is complete when its
// last result scores above everything the proof leaves unexamined; otherwise it is partial
// and carries that bound.
package provensearch
