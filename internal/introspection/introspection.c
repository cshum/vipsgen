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


// Helper function to get type name
char* get_type_name(GType type) {
    return (char*)g_type_name(type);
}

// Helper functions for type checking
int is_type_enum(GType type) {
    return G_TYPE_IS_ENUM(type);
}

int is_type_flags(GType type) {
    return G_TYPE_IS_FLAGS(type);
}

// Callback to collect argument information
static void collect_argument(VipsObject* object, GParamSpec* pspec,
                       VipsArgumentClass* argument_class,
                       VipsArgumentInstance* argument_instance,
                       ArgInfo* arg) {

    // Skip deprecated arguments
    if (argument_class->flags & VIPS_ARGUMENT_DEPRECATED)
        return;

    // Basic information
    arg->name = strdup(g_param_spec_get_name(pspec));
    arg->nick = strdup(g_param_spec_get_nick(pspec));
    arg->blurb = strdup(g_param_spec_get_blurb(pspec));
    arg->flags = argument_class->flags;
    arg->type_val = G_PARAM_SPEC_VALUE_TYPE(pspec);

    // Determine if input/output and required
    arg->is_input = (argument_class->flags & VIPS_ARGUMENT_INPUT) != 0;
    arg->is_output = (argument_class->flags & VIPS_ARGUMENT_OUTPUT) != 0;
    arg->required = (argument_class->flags & VIPS_ARGUMENT_REQUIRED) != 0;
}

// Structure to pass data to the callback
typedef struct {
    ArgInfo *args;
    int *count;
    int max_count;
} CollectArgsData;

// Callback wrapper for vips_argument_map
static void* collect_arguments_cb(VipsObject* object, GParamSpec* pspec,
                               VipsArgumentClass* argument_class,
                               VipsArgumentInstance* argument_instance,
                               void* a, void* b) {
    CollectArgsData *data = (CollectArgsData*)a;

    // Check if we have space for another argument
    if (*data->count >= data->max_count)
        return NULL;

    // Skip deprecated arguments
    if (argument_class->flags & VIPS_ARGUMENT_DEPRECATED)
        return NULL;

    // Collect argument info
    collect_argument(object, pspec, argument_class, argument_instance, &data->args[*data->count]);

    // Increment count
    (*data->count)++;

    return NULL;
}

ArgInfo* get_operation_arguments(const char *operation_name, int *count) {
    VipsOperation *op = vips_operation_new(operation_name);
    if (!op) {
        *count = 0;
        return NULL;
    }

    // Allocate space for a reasonable number of arguments
    const int max_args = 50;
    ArgInfo *args = (ArgInfo*)malloc(max_args * sizeof(ArgInfo));
    *count = 0;

    // Setup data for callback
    CollectArgsData data = {
        .args = args,
        .count = count,
        .max_count = max_args
    };

    // Map over all arguments
    vips_argument_map(VIPS_OBJECT(op), collect_arguments_cb, &data, NULL);

    g_object_unref(op);
    return args;
}

void free_operation_arguments(ArgInfo *args, int count) {
    if (!args) return;

    for (int i = 0; i < count; i++) {
        free(args[i].name);
        free(args[i].nick);
        free(args[i].blurb);
    }

    free(args);
}
