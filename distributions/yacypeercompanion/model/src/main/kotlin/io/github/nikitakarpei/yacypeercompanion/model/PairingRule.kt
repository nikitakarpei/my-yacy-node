package io.github.nikitakarpei.yacypeercompanion.model

fun pairingAllowed(permissionsGranted: Boolean, dataSharingAgreed: Boolean): Boolean = permissionsGranted && dataSharingAgreed
