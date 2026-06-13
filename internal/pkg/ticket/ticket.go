package ticket

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

func GenerateTicketRef() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return fmt.Sprintf("TIC-%d", n.Int64()+100000)
}
