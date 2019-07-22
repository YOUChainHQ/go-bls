package go_bls

import (
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/phoreproject/bls/g2pubs"
	"log"
)

type blsManager struct {
}

func NewBlsManager() BlsManager {
	return &blsManager{}
}

// GenerateKey generates a fresh key-pair for BLS signatures.
func (mgr *blsManager) GenerateKey() (SecretKey, PublicKey) {
	sk, err := g2pubs.RandKey(rand.Reader)
	if err != nil {
		log.Fatal("Can't generate secret key", err)
	}
	s := &secret{sk: sk}
	p, _ := s.PubKey()
	return s, p
}

//Aggregate aggregates signatures together into a new signature.
func (mgr *blsManager) Aggregate(sigs []Signature) (Signature, error) {
	switch l := len(sigs); l {
	case 0:
		return Signature{}, errors.New("no signatures")
	case 1:
		return sigs[0], nil
	default:
		g1sigs := make([]*g2pubs.Signature, 0, l)
		for i, sig := range sigs {
			g1sig, err := g2pubs.DeserializeSignature(sig)
			if err != nil {
				return Signature{}, fmt.Errorf("find at lease one uncrrect signature, first index: %d, error: %v", i, err)
			}
			g1sigs = append(g1sigs, g1sig)
		}
		result := g2pubs.AggregateSignatures(g1sigs)
		return result.Serialize(), nil
	}
}

//AggregatePublic aggregates public keys together into a new PublicKey.
func (mgr *blsManager) AggregatePublic(pubs []PublicKey) (PublicKey, error) {
	switch l := len(pubs); l {
	case 0:
		return nil, errors.New("no keys to aggregate")
	default:
		//blank public key
		zeropk := g2pubs.NewAggregatePubkey()
		newPk := PublicKey(&public{pk: zeropk})
		for i, p := range pubs {
			err := newPk.Aggregate(p)
			if err != nil {
				return nil, fmt.Errorf("error when aggregating public keys. index: %d, error: %v", i, err)
			}
		}
		return newPk, nil
	}
}

// VerifyAggregatedOne verifies each public key against a message.
func (mgr *blsManager) VerifyAggregatedOne(pubs []PublicKey, m Message, sig Signature) error {
	originPubs, err := converPublicKeysToOrigin(pubs)
	if err != nil {
		return err
	}
	g1sig, err := g2pubs.DeserializeSignature(sig)
	if err != nil {
		return err
	}
	ok := g1sig.VerifyAggregateCommon(originPubs, m)
	if ok {
		return nil
	}
	return ErrSigMismatch
}

// VerifyAggregatedN verifies each public key against each message.
func (mgr *blsManager) VerifyAggregatedN(pubs []PublicKey, ms []Message, sig Signature) error {
	originPubs, err := converPublicKeysToOrigin(pubs)
	if err != nil {
		return err
	}
	g1sig, err := g2pubs.DeserializeSignature(sig)
	if err != nil {
		return err
	}
	if len(originPubs) != len(ms) {
		return fmt.Errorf("different length of pubs and messages, %d vs %d", len(originPubs), len(ms))
	}
	msgs := make([][]byte, len(ms))
	for i, m := range ms {
		msgs[i] = m
	}
	ok := g1sig.VerifyAggregate(originPubs, msgs)
	if ok {
		return nil
	}
	return ErrSigMismatch
}

//DecompressPublicKey
func (mgr *blsManager) DecompressPublicKey(b CompressedPublic) (PublicKey, error) {
	pk, err := g2pubs.DeserializePublicKey(b)
	return &public{pk: pk}, err
}

//DecompressPrivateKey
func (mgr *blsManager) DecompressPrivateKey(b CompressedSecret) (SecretKey, error) {
	sk := g2pubs.DeserializeSecretKey(b)
	if sk == nil {
		return nil, errors.New("invalid secret key bytes")
	}
	return &secret{sk: sk}, nil
}

func converPublicKeysToOrigin(pubs []PublicKey) ([]*g2pubs.PublicKey, error) {
	origins := make([]*g2pubs.PublicKey, 0, len(pubs))
	for i, p := range pubs {
		gp, ok := p.(*public)
		if !ok {
			return origins, fmt.Errorf("invalid public key, index: %d", i)
		}
		origins = append(origins, gp.pk)
	}
	return origins, nil
}