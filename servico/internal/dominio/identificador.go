package dominio

import (
	"fmt"
	"io"
	"regexp"
	"time"
)

// alfabetoUlid é o Crockford base32 usado por ULIDs.
const alfabetoUlid = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

var padraoUlid = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)

// NovoUlid gera um ULID (26 caracteres) a partir do instante dado e de dez
// bytes aleatórios. É ordenável por tempo e não exige bibliotecas externas.
func NovoUlid(instante time.Time, aleatorio io.Reader) (string, error) {
	var id [16]byte
	ms := uint64(instante.UnixMilli())
	id[0] = byte(ms >> 40)
	id[1] = byte(ms >> 32)
	id[2] = byte(ms >> 24)
	id[3] = byte(ms >> 16)
	id[4] = byte(ms >> 8)
	id[5] = byte(ms)
	if _, err := io.ReadFull(aleatorio, id[6:]); err != nil {
		return "", fmt.Errorf("gerar entropia do identificador: %w", err)
	}
	return codificarUlid(id), nil
}

// UlidValido informa se s tem a forma de um ULID.
func UlidValido(s string) bool { return padraoUlid.MatchString(s) }

func codificarUlid(id [16]byte) string {
	var dst [26]byte
	dst[0] = alfabetoUlid[(id[0]&224)>>5]
	dst[1] = alfabetoUlid[id[0]&31]
	dst[2] = alfabetoUlid[(id[1]&248)>>3]
	dst[3] = alfabetoUlid[((id[1]&7)<<2)|((id[2]&192)>>6)]
	dst[4] = alfabetoUlid[(id[2]&62)>>1]
	dst[5] = alfabetoUlid[((id[2]&1)<<4)|((id[3]&240)>>4)]
	dst[6] = alfabetoUlid[((id[3]&15)<<1)|((id[4]&128)>>7)]
	dst[7] = alfabetoUlid[(id[4]&124)>>2]
	dst[8] = alfabetoUlid[((id[4]&3)<<3)|((id[5]&224)>>5)]
	dst[9] = alfabetoUlid[id[5]&31]
	dst[10] = alfabetoUlid[(id[6]&248)>>3]
	dst[11] = alfabetoUlid[((id[6]&7)<<2)|((id[7]&192)>>6)]
	dst[12] = alfabetoUlid[(id[7]&62)>>1]
	dst[13] = alfabetoUlid[((id[7]&1)<<4)|((id[8]&240)>>4)]
	dst[14] = alfabetoUlid[((id[8]&15)<<1)|((id[9]&128)>>7)]
	dst[15] = alfabetoUlid[(id[9]&124)>>2]
	dst[16] = alfabetoUlid[((id[9]&3)<<3)|((id[10]&224)>>5)]
	dst[17] = alfabetoUlid[id[10]&31]
	dst[18] = alfabetoUlid[(id[11]&248)>>3]
	dst[19] = alfabetoUlid[((id[11]&7)<<2)|((id[12]&192)>>6)]
	dst[20] = alfabetoUlid[(id[12]&62)>>1]
	dst[21] = alfabetoUlid[((id[12]&1)<<4)|((id[13]&240)>>4)]
	dst[22] = alfabetoUlid[((id[13]&15)<<1)|((id[14]&128)>>7)]
	dst[23] = alfabetoUlid[(id[14]&124)>>2]
	dst[24] = alfabetoUlid[((id[14]&3)<<3)|((id[15]&224)>>5)]
	dst[25] = alfabetoUlid[id[15]&31]
	return string(dst[:])
}
