package io.github.nikitakarpei.yacypeercompanion.services

import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.Log
import pairing.DeviceKey
import java.security.KeyPairGenerator
import java.security.KeyStore
import java.security.PrivateKey
import java.security.Signature
import java.security.spec.ECGenParameterSpec

private const val ALIAS = "pairing"
private const val KEYSTORE = "AndroidKeyStore"
private const val LOG_TAG = "PairingKey"

object PairingKey {
    private val keyStore by lazy { KeyStore.getInstance(KEYSTORE).apply { load(null) } }

    @Synchronized fun generate(challenge: ByteArray) {
        try {
            generateKey(challenge, strongBox = true)
        } catch (failure: Exception) {
            Log.w(LOG_TAG, "pairing key not in StrongBox", failure)
            keyStore.deleteEntry(ALIAS)
            generateKey(challenge, strongBox = false)
        }
    }

    @Synchronized fun attestationChain(): List<ByteArray> = keyStore.getCertificateChain(ALIAS).map { it.encoded }

    @Synchronized fun current(): DeviceKey? {
        val privateKey = keyStore.getKey(ALIAS, null) as? PrivateKey ?: return null
        val certificate = keyStore.getCertificate(ALIAS) ?: return null
        return KeystoreDeviceKey(privateKey, certificate.publicKey.encoded)
    }

    @Synchronized fun delete() {
        keyStore.deleteEntry(ALIAS)
    }

    private fun generateKey(challenge: ByteArray, strongBox: Boolean) {
        val spec = KeyGenParameterSpec.Builder(ALIAS, KeyProperties.PURPOSE_SIGN)
            .setAlgorithmParameterSpec(ECGenParameterSpec("secp256r1"))
            .setDigests(KeyProperties.DIGEST_NONE, KeyProperties.DIGEST_SHA256)
            .setAttestationChallenge(challenge)
            .setIsStrongBoxBacked(strongBox)
            .build()
        KeyPairGenerator.getInstance(KeyProperties.KEY_ALGORITHM_EC, KEYSTORE).apply { initialize(spec) }.generateKeyPair()
    }
}

private class KeystoreDeviceKey(private val privateKey: PrivateKey, private val publicKeyInfo: ByteArray) : DeviceKey {
    override fun publicKeyInfo(): ByteArray = publicKeyInfo

    override fun signDigest(digest: ByteArray): ByteArray = Signature.getInstance("NONEwithECDSA").run {
        initSign(privateKey)
        update(digest)
        sign()
    }
}
