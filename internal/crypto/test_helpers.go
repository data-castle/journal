package crypto

import "filippo.io/age"

// generateRecipients generates n valid age recipients for testing
func generateRecipients(n int) []string {
	var recipients []string
	for range n {
		identity, _ := age.GenerateX25519Identity()
		recipients = append(recipients, identity.Recipient().String())
	}
	return recipients
}
