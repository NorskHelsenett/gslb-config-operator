package controller

import "time"

const (
	requeueAfter time.Duration = time.Minute * 30

	conditionAccepted   = "Accepted"
	conditionProgrammed = "Programmed"

	gslbServiceFinalizer = "lb.nhn.no/finalizer"
)
