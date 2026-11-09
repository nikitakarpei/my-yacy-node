package io.github.nikitakarpei.yacypeercompanion.services

import io.github.nikitakarpei.yacypeercompanion.model.PairLink
import pairingrequests.Pairingrequests

fun pairLinkFrom(text: String): PairLink? = text.takeIf { Pairingrequests.linkKindOf(it) == Pairingrequests.LinkPair }?.let { PairLink(it) }

fun needsNewerApp(text: String): Boolean = Pairingrequests.linkKindOf(text) == Pairingrequests.LinkUnknownKind
