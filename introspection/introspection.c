#include "introspection.h"

static void* collect_operations(void *object_class, void *a, void *b) {
    OperationList *list = (OperationList *)a;
    VipsObjectClass *vobject_class = VIPS_OBJECT_CLASS(object_class);

    if (vobject_class && vobject_class->nickname &&
        G_TYPE_CHECK_CLASS_TYPE(vobject_class, VIPS_TYPE_OPERATION)) {
        const char *nickname = vobject_class->nickname;

        if (list->count >= list->capacity) {
            list->capacity *= 2;
            list->names = realloc(list->names, list->capacity * sizeof(char*));
        }
        list->names[list->count] = strdup(nickname);
        list->count++;
    }

    return NULL;
}

// This function discovers all operations by directly querying GType system
char** get_all_operation_names(int *count) {
    OperationList list = {
        .names = malloc(1000 * sizeof(char*)),
        .count = 0,
        .capacity = 1000
    };

    // Get all types derived from VipsOperation
    GType base_type = VIPS_TYPE_OPERATION;
    guint n_children = 0;
    GType *children = g_type_children(base_type, &n_children);

    // Process each child type
    for (guint i = 0; i < n_children; i++) {
        // Only process non-abstract types
        if (!G_TYPE_IS_ABSTRACT(children[i])) {
            // Get the class to access the nickname
            VipsObjectClass *class = VIPS_OBJECT_CLASS(g_type_class_ref(children[i]));

            if (class && class->nickname) {
                // Check if we can actually instantiate this operation
                VipsOperation *op = VIPS_OPERATION(g_object_new(children[i], NULL));
                if (op) {
                    // Expand array if needed
                    if (list.count >= list.capacity) {
                        list.capacity *= 2;
                        list.names = realloc(list.names, list.capacity * sizeof(char*));
                    }

                    // Add the operation name
                    list.names[list.count++] = strdup(class->nickname);
                    g_object_unref(op);
                }
            }

            g_type_class_unref(class);
        }

        // Some operations might have their own child types
        guint n_grandchildren = 0;
        GType *grandchildren = g_type_children(children[i], &n_grandchildren);

        for (guint j = 0; j < n_grandchildren; j++) {
            if (!G_TYPE_IS_ABSTRACT(grandchildren[j])) {
                VipsObjectClass *class = VIPS_OBJECT_CLASS(g_type_class_ref(grandchildren[j]));

                if (class && class->nickname) {
                    VipsOperation *op = VIPS_OPERATION(g_object_new(grandchildren[j], NULL));
                    if (op) {
                        if (list.count >= list.capacity) {
                            list.capacity *= 2;
                            list.names = realloc(list.names, list.capacity * sizeof(char*));
                        }

                        list.names[list.count++] = strdup(class->nickname);
                        g_object_unref(op);
                    }
                }

                g_type_class_unref(class);
            }
        }

        g_free(grandchildren);
    }

    g_free(children);

    *count = list.count;
    return list.names;
}

void free_operation_names(char **names, int count) {
    for (int i = 0; i < count; i++) {
        free(names[i]);
    }
    free(names);
}

EnumValueInfo* get_enum_values(const char *enum_type_name, int *count) {
    GType type = g_type_from_name(enum_type_name);
    if (type == 0) {
        *count = 0;
        return NULL;
    }

    // Get the enum class
    GEnumClass *enum_class = G_ENUM_CLASS(g_type_class_ref(type));
    if (!enum_class) {
        *count = 0;
        return NULL;
    }

    // Allocate space for values
    *count = enum_class->n_values;
    if (*count <= 0 || *count > 100) { // Sanity check
        g_type_class_unref(enum_class);
        *count = 0;
        return NULL;
    }

    EnumValueInfo *values = malloc(*count * sizeof(EnumValueInfo));
    if (!values) {
        g_type_class_unref(enum_class);
        *count = 0;
        return NULL;
    }

    // Copy the values with NULL checks
    for (int i = 0; i < *count; i++) {
        // Check for NULL pointers that might cause segfault
        if (enum_class->values[i].value_name) {
            values[i].name = strdup(enum_class->values[i].value_name);
        } else {
            values[i].name = strdup("UNKNOWN");
        }

        values[i].value = enum_class->values[i].value;

        if (enum_class->values[i].value_nick) {
            values[i].nick = strdup(enum_class->values[i].value_nick);
        } else {
            values[i].nick = strdup("");
        }
    }

    // Unref the class
    g_type_class_unref(enum_class);
    return values;
}

// Check if a type exists
int type_exists(const char *type_name) {
    GType type = g_type_from_name(type_name);
    return type != 0 ? 1 : 0;
}

// Free enum value info
void free_enum_values(EnumValueInfo *values, int count) {
    for (int i = 0; i < count; i++) {
        free(values[i].name);
        free(values[i].nick);
    }
    free(values);
}

GObjectClass* get_object_class(void* obj) {
    return G_OBJECT_GET_CLASS(obj);
}

static void *vips_object_find_args(VipsObject *object, GParamSpec *pspec,
                                  VipsArgumentClass *argument_class,
                                  VipsArgumentInstance *argument_instance,
                                  void *a, void *b) {
    VipsNameFlagsPair *pair = (VipsNameFlagsPair *)a;
    int *i = (int *)b;

    pair->names[*i] = g_param_spec_get_name(pspec);
    pair->flags[*i] = argument_class->flags;

    *i += 1;

    return NULL;
}

int get_vips_operation_args(VipsOperation *op, char ***names, int **flags, int *n_args) {
    VipsObject *object = VIPS_OBJECT(op);
    VipsObjectClass *object_class = VIPS_OBJECT_GET_CLASS(object);
    int n = g_slist_length(object_class->argument_table_traverse);

    *names = (char **)g_malloc(n * sizeof(char *));
    *flags = (int *)g_malloc(n * sizeof(int));
    if (!*names || !*flags)
        return -1;

    // Use vips_argument_map to collect arguments
    VipsNameFlagsPair pair = {
        .names = (const char **)(*names),
        .flags = *flags
    };
    int i = 0;
    vips_argument_map(object, vips_object_find_args, &pair, &i);

    if (n_args)
        *n_args = n;

    return 0;
}
