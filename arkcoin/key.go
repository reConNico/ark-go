package arkcoin

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"log"

	"github.com/kristjank/ark-go/arkcoin/base58"

	"github.com/btcsuite/btcd/btcec"
	"golang.org/x/crypto/ripemd160"
)

var (
	secp256k1 = btcec.S256()
)

//Params is parameters of the coin.
type Params struct {
	DumpedPrivateKeyHeader []byte
	AddressHeader          byte
	P2SHHeader             byte
	HDPrivateKeyID         []byte
	HDPublicKeyID          []byte
}

//PublicKey represents public key for bitcoin
type PublicKey struct {
	*btcec.PublicKey
	isCompressed bool
	param        *Params
}

//PrivateKey represents private key for bitcoin
type PrivateKey struct {
	*btcec.PrivateKey
	PublicKey *PublicKey
}

//NewPublicKey returns PublicKey struct using public key hex string.
func NewPublicKey(pubKeyByte []byte, param *Params) (*PublicKey, error) {
	key, err := btcec.ParsePubKey(pubKeyByte, secp256k1)
	if err != nil {
		return nil, err
	}
	isCompressed := false
	if len(pubKeyByte) == btcec.PubKeyBytesLenCompressed {
		isCompressed = true
	}
	return &PublicKey{
		PublicKey:    key,
		isCompressed: isCompressed,
		param:        param,
	}, nil
}

//FromWIF gets PublicKey and PrivateKey from private key of WIF format.
func FromWIF(wif string, param *Params) (*PrivateKey, error) {
	pb, err := base58.Decode(wif)
	if err != nil {
		return nil, err
	}
	ok := false
	for _, h := range param.DumpedPrivateKeyHeader {
		if pb[0] == h {
			ok = true
		}
	}
	if !ok {
		return nil, errors.New("wif is invalid")
	}
	isCompressed := false
	if len(pb) == btcec.PrivKeyBytesLen+2 && pb[btcec.PrivKeyBytesLen+1] == 0x01 {
		pb = pb[:len(pb)-1]
		isCompressed = true
		log.Println("compressed")
	}

	//Get the raw public
	priv, pub := btcec.PrivKeyFromBytes(secp256k1, pb[1:])
	return &PrivateKey{
		PrivateKey: priv,
		PublicKey: &PublicKey{
			PublicKey:    pub,
			isCompressed: isCompressed,
			param:        param,
		},
	}, nil
}

//NewPrivateKeyFromPassword creates and returns PrivateKey from string.
func NewPrivateKeyFromPassword(password string, param *Params) *PrivateKey {
	h := sha256.New()
	h.Write([]byte(password))
	pb := h.Sum(nil)

	//priv, pub := btcec.PrivKeyFromBytes(secp256k1, pb)
	return NewPrivateKey(pb, param)
	/* &PrivateKey{
		PrivateKey: priv,
		PublicKey: &PublicKey{
			PublicKey:    pub,
			isCompressed: true,
			param:        param,
		},
	}*/
}

//NewPrivateKey creates and returns PrivateKey from bytes.
func NewPrivateKey(pb []byte, param *Params) *PrivateKey {
	priv, pub := btcec.PrivKeyFromBytes(secp256k1, pb)
	return &PrivateKey{
		PrivateKey: priv,
		PublicKey: &PublicKey{
			PublicKey:    pub,
			isCompressed: true,
			param:        param,
		},
	}
}

//Generate generates random PublicKey and PrivateKey.
func Generate(param *Params) (*PrivateKey, error) {
	prikey, err := btcec.NewPrivateKey(secp256k1)
	if err != nil {
		return nil, err
	}
	key := &PrivateKey{
		PublicKey: &PublicKey{
			PublicKey:    prikey.PubKey(),
			isCompressed: true,
			param:        param,
		},
		PrivateKey: prikey,
	}

	return key, nil
}

//Sign sign data.
func (priv *PrivateKey) Sign(hash []byte) ([]byte, error) {
	sig, err := priv.PrivateKey.Sign(hash)
	if err != nil {
		return nil, err
	}
	return sig.Serialize(), nil
}

//WIFAddress returns WIF format string from PrivateKey
func (priv *PrivateKey) WIFAddress() string {
	p := priv.Serialize()
	if priv.PublicKey.isCompressed {
		p = append(p, 0x1)
	}
	p = append(p, 0x0)
	copy(p[1:], p[:len(p)-1])
	p[0] = priv.PublicKey.param.DumpedPrivateKeyHeader[0]
	return base58.Encode(p)
}

//Serialize serializes public key depending on isCompressed.
func (pub *PublicKey) Serialize() []byte {
	if pub.isCompressed {
		return pub.SerializeCompressed()
	}
	return pub.SerializeUncompressed()
}

//AddressBytes returns bitcoin address  bytes from PublicKey
func (pub *PublicKey) AddressBytes() []byte {
	//Next we get a sha256 hash of the public key generated
	//via ECDSA, and then get a ripemd160 hash of the sha256 hash.
	//shadPublicKeyBytes := sha256.Sum256(pub.Serialize())
	shadPublicKeyBytes := pub.Serialize()

	ripeHash := ripemd160.New()
	if _, err := ripeHash.Write(shadPublicKeyBytes[:]); err != nil {
		log.Fatal(err)
	}
	return ripeHash.Sum(nil)
}

//Address returns bitcoin address from PublicKey
func (pub *PublicKey) Address() string {
	ripeHashedBytes := pub.AddressBytes()
	ripeHashedBytes = append(ripeHashedBytes, 0x0)
	copy(ripeHashedBytes[1:], ripeHashedBytes[:len(ripeHashedBytes)-1])
	ripeHashedBytes[0] = pub.param.AddressHeader

	return base58.Encode(ripeHashedBytes)
}

//DecodeAddress converts bitcoin address to hex form.
func DecodeAddress(addr string) ([]byte, error) {
	pb, err := base58.Decode(addr)
	if err != nil {
		return nil, err
	}
	return pb[1:], nil
}

//Verify verifies signature is valid or not.
func (pub *PublicKey) Verify(signature []byte, data []byte) error {
	sig, err := btcec.ParseSignature(signature, secp256k1)
	if err != nil {
		return err
	}
	valid := sig.Verify(data, pub.PublicKey)
	if !valid {
		return fmt.Errorf("signature is invalid")
	}
	return nil
}

//AddressBytes returns ripeme160(sha256(redeem)) (address of redeem script).
func AddressBytes(redeem []byte) []byte {
	//h := sha256.Sum256(redeem)
	h := redeem
	ripeHash := ripemd160.New()
	if _, err := ripeHash.Write(h[:]); err != nil {
		log.Fatal(err)
	}
	return ripeHash.Sum(nil)
}

//Address returns ripeme160(sha256(redeem)) (address of redeem script).
func Address(redeem []byte, header byte) string {
	ripeHashedBytes := AddressBytes(redeem)
	ripeHashedBytes = append(ripeHashedBytes, 0x0)
	copy(ripeHashedBytes[1:], ripeHashedBytes[:len(ripeHashedBytes)-1])
	ripeHashedBytes[0] = header

	return base58.Encode(ripeHashedBytes)
}
