package dominio

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
)

// TamanhoMinimoChaveHmac é o menor tamanho aceito para a chave da contrassenha.
const TamanhoMinimoChaveHmac = 16

// DigitosContrassenha é o tamanho do código supervisor da Crane National 168.
const DigitosContrassenha = 4

// GeradorContrassenha deriva o código supervisor de quatro dígitos de uma
// máquina a partir de uma chave secreta e do contador de rotação (ADR-0002):
//
//	codigo = HMAC-SHA256(chave, idMaquina || ":" || contador)
//	         → quatro primeiros bytes como inteiro big-endian, módulo 10 000,
//	           com zeros à esquerda.
//
// O código nunca é armazenado: é recomputado a partir do contador.
type GeradorContrassenha struct {
	chave []byte
}

// NovoGeradorContrassenha valida a chave e cria o gerador.
func NovoGeradorContrassenha(chave []byte) (*GeradorContrassenha, error) {
	if len(chave) < TamanhoMinimoChaveHmac {
		return nil, errors.New("chave HMAC da contrassenha muito curta")
	}
	copia := make([]byte, len(chave))
	copy(copia, chave)
	return &GeradorContrassenha{chave: copia}, nil
}

// Derivar devolve o código corrente para (idMaquina, contador).
func (g *GeradorContrassenha) Derivar(idMaquina string, contador uint64) string {
	mac := hmac.New(sha256.New, g.chave)
	mac.Write([]byte(idMaquina))
	mac.Write([]byte(":"))
	mac.Write([]byte(strconv.FormatUint(contador, 10)))
	soma := mac.Sum(nil)
	return fmt.Sprintf("%0*d", DigitosContrassenha, binary.BigEndian.Uint32(soma[:4])%10000)
}
