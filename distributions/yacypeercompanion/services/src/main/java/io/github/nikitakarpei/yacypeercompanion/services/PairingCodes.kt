package io.github.nikitakarpei.yacypeercompanion.services

import connectlink.Connectlink
import io.github.nikitakarpei.yacypeercompanion.model.PairingCode

fun pairingCodeOf(text: String): PairingCode? = runCatching { Connectlink.linkFrom(text) }.getOrNull()?.let { PairingCode(text) }
