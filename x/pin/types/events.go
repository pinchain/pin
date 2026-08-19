package types

// x/pin module event types and attribute keys.
const (
	EventTypeRegisterPIN = "register_pin"
	EventTypeSendByPIN   = "send_by_pin"

	AttributeKeyPIN            = "pin"
	AttributeKeyOwner          = "owner"
	AttributeKeyCreationHeight = "creation_height"
	AttributeKeySender         = "sender"
	AttributeKeyRecipient      = "recipient"
	AttributeKeyRecipientPIN   = "recipient_pin"
	AttributeKeyAmount         = "amount"
)
