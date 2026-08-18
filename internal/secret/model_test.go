package secret

import "testing"

func TestValidType(t *testing.T) {
	if !ValidType(TypeLoginPassword) || !ValidType(TypeText) || !ValidType(TypeBinary) || !ValidType(TypeBankCard) {
		t.Fatal("valid types rejected")
	}
	if ValidType("nope") {
		t.Fatal("invalid accepted")
	}
}

func TestToDTO(t *testing.T) {
	dto := ToDTO(Secret{ID: "1", Type: TypeText, Title: "t", Ciphertext: []byte{1, 2}, Version: 2})
	if dto.ID != "1" || dto.Ciphertext == "" || dto.Version != 2 {
		t.Fatalf("%+v", dto)
	}
}
