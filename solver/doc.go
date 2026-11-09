// Package solver is the scheduling engine: it places tasks into the free time
// of a day and reports the tasks it could not place.
//
// The package is pure. It does no I/O and never reads the clock; the current
// time is always an input. The same input always produces the same schedule.
package solver
