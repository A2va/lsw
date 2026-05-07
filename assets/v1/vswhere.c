#include <stdio.h>
#include <string.h>

int main(int argc, char **argv) {
  int prop_path = 0;
  int format_json = 0;

  for (int i = 1; i < argc; i++) {
    if (_stricmp(argv[i], "-property") == 0 && i + 1 < argc) {
      if (_stricmp(argv[i + 1], "installationpath") == 0) {
        prop_path = 1;
      }
    }
    if (_stricmp(argv[i], "-format") == 0 && i + 1 < argc) {
      if (_stricmp(argv[i + 1], "json") == 0) {
        format_json = 1;
      }
    }
  }

  // SCENARIO 1: Tool wants raw JSON (Modern tools)
  if (format_json) {
    printf("[\n");
    printf("  {\n");
    printf("    \"instanceId\": \"MSVCWine\",\n");
    printf("    \"installationName\": \"VisualStudio/__VS_VER__.0\",\n");
    printf("    \"installationPath\": \"C:\\\\Program Files\\\\MSVC\",\n");
    printf("    \"installationVersion\": \"%s.0\",\n", "__VS_VER__");
    printf("    \"displayName\": \"Visual Studio MSVC-Wine Mock\",\n");
    printf("    \"isPrerelease\": false\n");
    printf("  }\n");
    printf("]\n");
    return 0;
  }

  // SCENARIO 2: Tool wants JUST the path (CMake fallback)
  if (prop_path) {
    printf("C:\\Program Files\\MSVC\n");
    return 0;
  }

  // SCENARIO 3: Tool wants text output (Rust cc-rs / find-msvc-tools)
  // MUST CONTAIN installationName, installationPath, installationVersion!
  printf("instanceId: MSVCWine\n");
  printf("installationName: VisualStudio/__VS_VER__.0\n");
  printf("installationPath: C:\\Program Files\\MSVC\n");
  printf("installationVersion: __VS_VER__.0\n");
  printf("displayName: Visual Studio MSVC-Wine Mock\n");

  return 0;
}
