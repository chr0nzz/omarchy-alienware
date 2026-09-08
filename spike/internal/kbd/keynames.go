package kbd

const KeyTableAvailable = false

const KeyIndexNote = "the reference SDK ships no key index to key name table for vid 0d62 (Darfon); AlienFX_SDK::Mappings loads names from a user-populated mappings.json that starts empty, and the only Darfon-specific hint in the reference is alienfx-cli defaulting an interactive naming wizard to 0x88 (136) lights; addressing here is by raw wire index only"

var KeyNames = map[uint8]string{}

func KeyName(index uint8) (string, bool) {
	name, ok := KeyNames[index]
	return name, ok
}
