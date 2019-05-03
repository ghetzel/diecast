#ifndef GO_RENDERERS_SASS_H
#define GO_RENDERERS_SASS_H

#include <string.h>
#include <sass/context.h>

// Forward declaration
struct Sass_Compiler;
struct Sass_Importer_Entry;

Sass_Import_List diecast_sass_importer(const char* url, Sass_Importer_Entry cb, struct Sass_Compiler *compiler);

#endif