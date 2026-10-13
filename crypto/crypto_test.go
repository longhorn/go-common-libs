package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncryptVolumeName(t *testing.T) {
	// A v2 volume already owns a dm device under its own name, so its LUKS
	// mapping must not collide with it.
	assert.Equal(t, "vol", EncryptVolumeName("vol", "v1"))
	assert.Equal(t, "vol", EncryptVolumeName("vol", ""))
	assert.Equal(t, "vol-encrypted", EncryptVolumeName("vol", "v2"))
}

func TestVolumeMapper(t *testing.T) {
	assert.Equal(t, "/dev/mapper/vol", VolumeMapper("vol", "v1"))
	assert.Equal(t, "/dev/mapper/vol-encrypted", VolumeMapper("vol", "v2"))
}

func TestEncryptParamsDefaults(t *testing.T) {
	p := NewEncryptParams("", "", "", "", "", "", "")
	assert.Equal(t, CryptoKeyDefaultCipher, p.GetKeyCipher())
	assert.Equal(t, CryptoKeyDefaultHash, p.GetKeyHash())
	assert.Equal(t, CryptoKeyDefaultSize, p.GetKeySize())
	assert.Equal(t, CryptoDefaultPBKDF, p.GetPBKDF())
	// The PBKDF tuning knobs stay empty so cryptsetup keeps its own defaults.
	assert.Equal(t, "", p.GetPBKDFForceIterations())
	assert.Equal(t, "", p.GetPBKDFMemory())
}

func TestEncryptParamsLuksFormatOptions(t *testing.T) {
	p := NewEncryptParams("vault", "aes-cbc-essiv:sha256", "sha512", "512", "pbkdf2", "1000", "256")
	options := p.LuksFormatOptions()
	assert.Equal(t, "aes-cbc-essiv:sha256", options.KeyCipher)
	assert.Equal(t, "sha512", options.KeyHash)
	assert.Equal(t, "512", options.KeySize)
	assert.Equal(t, "pbkdf2", options.PBKDF)
	assert.Equal(t, "1000", options.PBKDFForceIterations)
	assert.Equal(t, "256", options.PBKDFMemory)
}

func TestDeviceEncryptionStatusNonMapperPath(t *testing.T) {
	// A path outside /dev/mapper is returned unchanged, without calling cryptsetup.
	device, mapper, err := DeviceEncryptionStatus("/dev/longhorn/vol")
	assert.NoError(t, err)
	assert.Equal(t, "/dev/longhorn/vol", device)
	assert.Equal(t, "", mapper)
}
