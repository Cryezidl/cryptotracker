package model

import pb "cryptotracker/proto/rates"

type PairKey struct {
	Base  string
	Quote string
}

func NewPairKey(pbPair *pb.CurrencyPair) PairKey {
	if pbPair == nil {
		return PairKey{}
	}
	return PairKey{
		Base:  pbPair.GetBase(),
		Quote: pbPair.GetQuote(),
	}
}
