package mps

import (
	"math/rand"
	"time"
)

var (
	// global random numbers for MPS. Go v1.20
	mpsRand = rand.New(rand.NewSource(time.Now().UnixNano()))
)
