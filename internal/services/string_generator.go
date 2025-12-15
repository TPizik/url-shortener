package services

import (
	"math/rand"
)

type StringGenerator struct {
}

func NewStringGenerator() *StringGenerator {
	return &StringGenerator{}
}

func (sg *StringGenerator) GenerateRandom() string {
	return Base62Encode(rand.Uint64())
}
