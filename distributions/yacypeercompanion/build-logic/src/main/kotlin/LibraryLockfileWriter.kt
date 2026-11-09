import org.gradle.api.DefaultTask
import org.gradle.api.artifacts.component.ModuleComponentIdentifier
import org.gradle.api.artifacts.result.ResolvedComponentResult
import org.gradle.api.artifacts.result.ResolvedDependencyResult
import org.gradle.api.file.RegularFileProperty
import org.gradle.api.provider.Property
import org.gradle.api.tasks.Input
import org.gradle.api.tasks.OutputFile
import org.gradle.api.tasks.TaskAction

abstract class LibraryLockfileWriter : DefaultTask() {
    @get:Input
    abstract val rootComponent: Property<ResolvedComponentResult>

    @get:Input
    abstract val configurationName: Property<String>

    @get:OutputFile
    abstract val lockfile: RegularFileProperty

    @TaskAction
    fun write() {
        val lines = librariesOf(rootComponent.get()).sorted().map { "$it=${configurationName.get()}" } + "empty="
        lockfile.get().asFile.writeText(lines.joinToString("\n", postfix = "\n"))
    }
}

private fun librariesOf(root: ResolvedComponentResult): Set<String> {
    val components = mutableSetOf<ResolvedComponentResult>()
    val pending = ArrayDeque(listOf(root))
    while (pending.isNotEmpty()) {
        val component = pending.removeFirst()
        if (components.add(component)) {
            pending += component.dependencies.filterIsInstance<ResolvedDependencyResult>().map { it.selected }
        }
    }
    return components.mapNotNull { it.id as? ModuleComponentIdentifier }.map { "${it.group}:${it.module}:${it.version}" }.toSet()
}
