// Copied from attested-web-capture-android 0.2.0 (commit 75cc4a5); see its design notes there.
package io.github.nikitakarpei.attestedstatements

import org.json.JSONArray
import org.json.JSONObject
import java.security.MessageDigest
import java.security.cert.X509Certificate
import java.util.Base64

const val PROFILE = "android"

const val SIGNED_ID_PREFIX = "hardware-attested-statement/v0.1/signed"

const val DSSE_PAYLOAD_TYPE = "application/vnd.in-toto+json"

/** The Android profile's key evidence: the attestation chain, without the self-signed root (verifiers pin it). */
fun keyEvidence(chain: List<X509Certificate>): ByteArray = JSONObject()
    .put("profile", PROFILE)
    .put(
        "certificates",
        JSONArray(
            chain.filter {
                it.subjectX500Principal != it.issuerX500Principal
            }.map { Base64.getEncoder().encodeToString(it.encoded) },
        ),
    )
    .toString().toByteArray()

/** The leaf of a time-stamp tree: binds the statement, the profile, the key's binding (the leaf certificate's
 *  signed part, holding the key and its attestation), and the signature. */
fun signedStatementId(statement: ByteArray, keyBinding: ByteArray, signature: ByteArray): ByteArray = sha256(
    SIGNED_ID_PREFIX.toByteArray() + byteArrayOf(PROFILE.length.toByte()) + PROFILE.toByteArray() + sha256(statement) +
        sha256(keyBinding) +
        signature,
)

/** DSSE's pre-authentication encoding: binds the payload type into what's signed. */
fun pae(statement: ByteArray): ByteArray =
    "DSSEv1 ${DSSE_PAYLOAD_TYPE.length} $DSSE_PAYLOAD_TYPE ${statement.size} ".toByteArray() + statement

fun sha256(b: ByteArray): ByteArray = MessageDigest.getInstance("SHA-256").digest(b)

fun hex(b: ByteArray) = b.joinToString("") { "%02x".format(it) }

/** A temporary failure of the proxy itself, not of the target: 503 with Retry-After. */
class Retryable(message: String, cause: Throwable? = null) : Exception(message, cause)
