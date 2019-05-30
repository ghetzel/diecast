#include "renderers_sass.h"

// The callback function that is fired whenever an @import statement is encountered.
Sass_Import_List diecast_sass_importer(const char *url, Sass_Importer_Entry cb, struct Sass_Compiler *compiler) {
    void* cookie = sass_importer_get_cookie(cb);
    Sass_Import_List list = sass_make_import_list(1);

    char* data;

    // get the file contents
    if (go_retrievePath(cookie, url, &data) >= 0) {
        // success, return the data
        list[0] = sass_make_import_entry(url, data, 0);
    } else {
        // error, return the message
        list[0] = sass_make_import_entry(url, 0, 0);
        sass_import_set_error(list[0], data, 0, 0);
    }

    return list;
}
