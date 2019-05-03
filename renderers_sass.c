#include "renderers_sass.h"

bool hasPrefix(const char *str, const char *pre) {
    size_t lenpre = strlen(pre);
    size_t lenstr = strlen(str);

    if (lenstr < lenpre) {
        return false;
    } else {
        return strncmp(pre, str, lenpre) == 0;
    }
}


Sass_Import_List diecast_sass_importer(const char *url, Sass_Importer_Entry cb, struct Sass_Compiler *compiler) {
    void* cookie = sass_importer_get_cookie(cb);
    Sass_Import_List list = sass_make_import_list(1);

    const char* data;

    // get the remote file contents
    if (go_retrievePath(cookie, url, &data) >= 0) {
        list[0] = sass_make_import_entry(url, sass_copy_c_string(data), 0);
    } else {
        list[0] = sass_make_import_entry(url, 0, 0);
        sass_import_set_error(list[0], sass_copy_c_string(data), 0, 0);
    }

    return list;
}
