package io.github.nikitakarpei.yacypeercompanion.services

import io.github.nikitakarpei.yacypeercompanion.model.PairingCode
import pairing.Pairing

fun pairingCodeOf(text: String): PairingCode? = text.takeIf { Pairing.linkKindOf(it) == Pairing.LinkPair }?.let { PairingCode(it) }

fun needsNewerApp(text: String): Boolean = Pairing.linkKindOf(text) == Pairing.LinkUnreadable
