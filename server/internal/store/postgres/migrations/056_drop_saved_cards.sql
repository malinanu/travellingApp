-- Saved cards were collected by sending the full card number to this
-- server, which only kept the brand and last four digits: no gateway token
-- was ever stored, so the rows cannot be charged and only imply a payment
-- capability that never existed. Payment now happens on the gateway's
-- hosted checkout, and card details never reach Spotto.
DELETE FROM payment_methods;
