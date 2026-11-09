package io.github.nikitakarpei.yacypeercompanion

import android.content.Context
import io.github.nikitakarpei.yacypeercompanion.model.pairingAllowed

object PairingPrerequisites {
    fun met(ctx: Context): Boolean = pairingAllowed(Permission.lastingGranted(ctx), DataSharing.agreed(ctx))
}
