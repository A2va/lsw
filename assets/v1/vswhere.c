#include <stdio.h>

int main(void) {
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
