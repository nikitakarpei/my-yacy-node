// Copied from attested-web-capture-android 0.2.0 (commit 75cc4a5); see its design notes there.
package io.github.nikitakarpei.attestedstatements

import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import org.bouncycastle.asn1.ASN1Integer
import org.bouncycastle.asn1.ASN1Sequence
import org.bouncycastle.asn1.DERSequence
import java.math.BigInteger
import java.security.KeyPairGenerator
import java.security.KeyStore
import java.security.PrivateKey
import java.security.Signature
import java.security.cert.X509Certificate
import java.security.spec.ECGenParameterSpec

const val KEY_CHALLENGE = "hardware-attested-statement/v0.1/android/key"

const val DAY_MS = 24 * 3600 * 1000L

class SigningKey(val privateKey: PrivateKey, val chain: List<X509Certificate>, val storage: String) {
    fun sign(data: ByteArray): ByteArray = lowS(
        Signature.getInstance("SHA256withECDSA").run {
            initSign(privateKey)
            update(data)
            sign()
        },
    )
}

val P256_ORDER = BigInteger("FFFFFFFF00000000FFFFFFFFFFFFFFFFBCE6FAADA7179E84F3B9CAC2FC632551", 16)

/** (r, n − s) is as valid as (r, s); the spec requires the smaller s, so each signing has exactly one ID. */
fun lowS(der: ByteArray): ByteArray {
    val seq = ASN1Sequence.getInstance(der)
    val r = (seq.getObjectAt(0) as ASN1Integer).value
    val s = (seq.getObjectAt(1) as ASN1Integer).value
    val low = if (s > P256_ORDER.shiftRight(1)) P256_ORDER - s else s
    return DERSequence(arrayOf(ASN1Integer(r), ASN1Integer(low))).encoded
}

object Keys {
    private val ks by lazy { KeyStore.getInstance("AndroidKeyStore").apply { load(null) } }

    @Volatile var current: SigningKey? = null
        private set

    @Synchronized fun get(): SigningKey {
        current?.let { k -> if (k.chain.all { it.notAfter.time - System.currentTimeMillis() > DAY_MS }) return k }
        val alias = "signing-" + System.currentTimeMillis()
        val storage = try {
            generate(alias, strongBox = true)
            "StrongBox"
        } catch (e: Exception) {
            runCatching { ks.deleteEntry(alias) }
            generate(alias, strongBox = false)
            "TEE"
        }
        val key = SigningKey(
            ks.getKey(alias, null) as PrivateKey,
            ks.getCertificateChain(alias).map { it as X509Certificate },
            storage,
        )
        current = key
        return key
    }

    @Synchronized fun reset() {
        current = null
        ks.aliases().toList().forEach { ks.deleteEntry(it) }
    }

    private fun generate(alias: String, strongBox: Boolean) {
        val spec = KeyGenParameterSpec.Builder(alias, KeyProperties.PURPOSE_SIGN)
            .setAlgorithmParameterSpec(ECGenParameterSpec("secp256r1"))
            .setDigests(KeyProperties.DIGEST_SHA256)
            .setAttestationChallenge(KEY_CHALLENGE.toByteArray())
            .setIsStrongBoxBacked(strongBox)
            .build()
        KeyPairGenerator.getInstance(KeyProperties.KEY_ALGORITHM_EC, "AndroidKeyStore")
            .apply { initialize(spec) }.generateKeyPair()
    }
}
