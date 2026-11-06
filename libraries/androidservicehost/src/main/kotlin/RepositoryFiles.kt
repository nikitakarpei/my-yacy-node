import org.gradle.api.Project
import java.io.File

fun Project.repositoryFile(path: String): File = rootProject.projectDir.resolve("../../$path").normalize()

fun Project.pinnedVersionOf(tool: String): String = repositoryFile("tools/tools.lock")
    .readLines()
    .map { it.trim().split(Regex("\\s+")) }
    .first { it[0] == tool }[1]
