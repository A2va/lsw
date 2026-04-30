#include <stdio.h>
#include <string.h>

int main(int argc, char **argv) {
  int prop_path = 0;
  char *find_target = NULL;

  for (int i = 1; i < argc; i++) {
    // Use case-insensitive _stricmp to satisfy xmake's lowercase
    // "installationpath"
    if (_stricmp(argv[i], "-property") == 0 && i + 1 < argc) {
      if (_stricmp(argv[i + 1], "installationpath") == 0) {
        prop_path = 1;
      }
    }
    if (_stricmp(argv[i], "-find") == 0 && i + 1 < argc) {
      find_target = argv[i + 1];
    }
  }

  // SCENARIO 1: Tool wants a specific executable
  if (find_target) {
    printf("C:\\Program Files\\MSVC\\%s\n", find_target);
    return 0;
  }

  // SCENARIO 2: Tool (like xmake) wants the raw text path
  if (prop_path) {
    printf("C:\\Program Files\\MSVC\n");
    return 0;
  }

  // SCENARIO 3: Tool (like CMake) wants JSON
  printf("[\n");
  printf("  {\n");
  printf("    \"instanceId\": \"MSVCWine\",\n");
  printf("    \"installationPath\": \"C:\\\\Program Files\\\\MSVC\",\n");
  printf("    \"installationVersion\": \"%s.0.0\",\n", "__VS_VER__");
  printf("    \"displayName\": \"Visual Studio MSVC-Wine Mock\",\n");
  printf("    \"isPrerelease\": false\n");
  printf("  }\n");
  printf("]\n");

  return 0;
}
