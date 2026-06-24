package main

// Sign tokens with private key for testing
import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)


type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func main() {
	// Read private.pem bytes using os.ReadFile
	data, err := os.ReadFile("certs/private.pem")
	if err != nil {
		fmt.Println(err)
		return 
	}
	

	// Parse the private key bytes 
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(data)
	if err != nil {
		fmt.Println(err)
		return 
	}
	
	// 3. Create your payload claims instance
	claims := Claims{
		Role:  "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: "user123",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // valid for 24 hours
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// 4. Initialize a token struct specifying RS256 signing method
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	// 5. Sign the token with your private key to get the final string
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		panic(err)
	}

	fmt.Println("Your Test Token:")
	fmt.Println(tokenString)
}