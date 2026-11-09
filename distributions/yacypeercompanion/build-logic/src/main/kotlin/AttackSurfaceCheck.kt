import org.gradle.api.DefaultTask
import org.gradle.api.GradleException
import org.gradle.api.file.DirectoryProperty
import org.gradle.api.file.RegularFileProperty
import org.gradle.api.provider.Property
import org.gradle.api.tasks.Input
import org.gradle.api.tasks.InputDirectory
import org.gradle.api.tasks.InputFile
import org.gradle.api.tasks.PathSensitive
import org.gradle.api.tasks.PathSensitivity
import org.gradle.api.tasks.TaskAction
import org.w3c.dom.Element
import java.io.File
import java.util.zip.ZipFile
import javax.xml.parsers.DocumentBuilderFactory

private const val ANDROID_NAMESPACE = "http://schemas.android.com/apk/res/android"
private const val WEB_VIEW_TYPE = "Landroid/webkit/WebView;"
private val PERMISSION_TAGS = listOf("uses-permission", "uses-permission-sdk-23")
private val COMPONENT_TAGS = listOf("activity", "activity-alias", "service", "receiver", "provider")
private val DEX_ENTRY = Regex("classes\\d*\\.dex")

abstract class AttackSurfaceCheck : DefaultTask() {
    @get:InputFile
    @get:PathSensitive(PathSensitivity.NONE)
    abstract val appManifest: RegularFileProperty

    @get:InputFile
    @get:PathSensitive(PathSensitivity.NONE)
    abstract val apkManifest: RegularFileProperty

    @get:InputDirectory
    @get:PathSensitive(PathSensitivity.NONE)
    abstract val apkDirectory: DirectoryProperty

    @get:Input
    abstract val namespace: Property<String>

    @TaskAction
    fun check() {
        val declared = surfaceOf(appManifest.get().asFile, namespace.get())
        val packaged = surfaceOf(apkManifest.get().asFile, namespace.get())
        val breaches = additionsOfLibraries(declared, packaged) + missingProtectionsOf(packaged) +
            webViewsIn(apkDirectory.get().asFile)
        if (breaches.isNotEmpty()) {
            throw GradleException("The release APK widens its attack surface:\n" + breaches.joinToString("\n"))
        }
    }
}

private class Surface(val permissions: Set<String>, val exportedComponents: Set<String>, val application: Element)

private fun surfaceOf(manifest: File, namespace: String): Surface {
    val root = DocumentBuilderFactory.newInstance().apply { isNamespaceAware = true }
        .newDocumentBuilder().parse(manifest).documentElement
    val permissions = PERMISSION_TAGS.flatMap { elementsOf(root, it) }.map { androidAttributeOf(it, "name") }
    val exportedComponents = COMPONENT_TAGS.flatMap { elementsOf(root, it) }
        .filter { androidAttributeOf(it, "exported") == "true" }
        .map { qualifiedNameOf(androidAttributeOf(it, "name"), namespace) }
    return Surface(permissions.toSet(), exportedComponents.toSet(), elementsOf(root, "application").single())
}

private fun elementsOf(root: Element, tag: String): List<Element> {
    val nodes = root.getElementsByTagName(tag)
    return (0 until nodes.length).map { nodes.item(it) as Element }
}

private fun androidAttributeOf(element: Element, name: String): String = element.getAttributeNS(ANDROID_NAMESPACE, name)

private fun qualifiedNameOf(name: String, namespace: String): String = when {
    name.startsWith(".") -> namespace + name
    "." !in name -> "$namespace.$name"
    else -> name
}

private fun additionsOfLibraries(declared: Surface, packaged: Surface): List<String> =
    (packaged.permissions - declared.permissions).map { "A library asks for the permission $it." } +
        (packaged.exportedComponents - declared.exportedComponents).map { "A library exports $it to other apps." }

private fun missingProtectionsOf(packaged: Surface): List<String> = listOfNotNull(
    "The app allows backups.".takeIf { androidAttributeOf(packaged.application, "allowBackup") != "false" },
    "The app allows cleartext traffic.".takeIf { androidAttributeOf(packaged.application, "usesCleartextTraffic") != "false" },
    "The app lets a debugger connect.".takeIf { androidAttributeOf(packaged.application, "debuggable") == "true" },
)

private fun webViewsIn(apkDirectory: File): List<String> = apkDirectory.listFiles { file -> file.extension == "apk" }.orEmpty()
    .flatMap { apk -> dexEntriesWithWebViewsIn(apk).map { "${apk.name}: $it uses a WebView." } }

private fun dexEntriesWithWebViewsIn(apk: File): List<String> = ZipFile(apk).use { zip ->
    zip.entries().asSequence().filter { DEX_ENTRY.matches(it.name) }
        .filter { String(zip.getInputStream(it).readBytes(), Charsets.ISO_8859_1).contains(WEB_VIEW_TYPE) }
        .map { it.name }.toList()
}
