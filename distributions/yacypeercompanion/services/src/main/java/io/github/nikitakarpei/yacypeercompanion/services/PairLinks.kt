package io.github.nikitakarpei.yacypeercompanion.services

import io.github.nikitakarpei.yacypeercompanion.model.PairLink
import pairing.Pairing

fun pairLinkFrom(text: String): PairLink? = text.takeIf { Pairing.linkKindOf(it) == Pairing.LinkPair }?.let { PairLink(it) }

fun needsNewerApp(text: String): Boolean = Pairing.linkKindOf(text) == Pairing.LinkUnreadable
