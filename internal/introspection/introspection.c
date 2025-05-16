#include "introspection.h"

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

    // Initialize default value fields
    arg->has_default = 0;
    arg->default_type = 0;
    arg->bool_default = FALSE;
    arg->int_default = 0;
    arg->double_default = 0.0;
    arg->string_default = NULL;
    arg->is_image = 0;
    arg->is_buffer = 0;
    arg->is_array = 0;

    // Determine parameter types
    GType type = arg->type_val;
    const char *name = arg->name;

    // Check if this is an image parameter
    if (g_type_is_a(type, vips_image_get_type())) {
        arg->is_image = 1;
    }

    // Check if this is a buffer parameter
    if ((strcmp(name, "buf") == 0 || strcmp(name, "buffer") == 0) &&
        (g_type_is_a(type, G_TYPE_POINTER) ||
         g_type_is_a(type, G_TYPE_BYTES) ||
         g_type_is_a(type, vips_blob_get_type()) ||
         g_type_name(type) == NULL ||
         strcmp(g_type_name(type), "gpointer") == 0)) {
        arg->is_buffer = 1;
    }

    // Check if this is an array parameter
    if (g_type_is_a(type, vips_array_double_get_type()) ||
        g_type_is_a(type, vips_array_int_get_type()) ||
        g_type_is_a(type, vips_array_image_get_type()) ||
        // Also check for pointer types used as arrays
        (g_type_is_a(type, G_TYPE_POINTER) &&
          (strcmp(name, "vector") == 0 || strcmp(name, "out_array") == 0)) ||
        // Check for common array parameter names with pointer types
        (g_type_is_a(type, G_TYPE_POINTER) &&
          (strcmp(name, "a") == 0 || strcmp(name, "b") == 0 ||
           strcmp(name, "c") == 0 || strcmp(name, "ink") == 0 ||
           strcmp(name, "coefficients") == 0))) {
        arg->is_array = 1;
    }

    // Get default value based on type
    if (G_IS_PARAM_SPEC_BOOLEAN(pspec)) {
        GParamSpecBoolean *pspec_bool = G_PARAM_SPEC_BOOLEAN(pspec);
        arg->has_default = 1;
        arg->default_type = 1;  // bool
        arg->bool_default = pspec_bool->default_value;
    }
    else if (G_IS_PARAM_SPEC_INT(pspec)) {
        GParamSpecInt *pspec_int = G_PARAM_SPEC_INT(pspec);
        arg->has_default = 1;
        arg->default_type = 2;  // int
        arg->int_default = pspec_int->default_value;
    }
    else if (G_IS_PARAM_SPEC_UINT(pspec)) {
        GParamSpecUInt *pspec_uint = G_PARAM_SPEC_UINT(pspec);
        arg->has_default = 1;
        arg->default_type = 2;  // int
        arg->int_default = (gint)pspec_uint->default_value;
    }
    else if (G_IS_PARAM_SPEC_DOUBLE(pspec)) {
        GParamSpecDouble *pspec_double = G_PARAM_SPEC_DOUBLE(pspec);
        arg->has_default = 1;
        arg->default_type = 3;  // double
        arg->double_default = pspec_double->default_value;
    }
    else if (G_IS_PARAM_SPEC_FLOAT(pspec)) {
        GParamSpecFloat *pspec_float = G_PARAM_SPEC_FLOAT(pspec);
        arg->has_default = 1;
        arg->default_type = 3;  // double
        arg->double_default = (gdouble)pspec_float->default_value;
    }
    else if (G_IS_PARAM_SPEC_STRING(pspec)) {
        GParamSpecString *pspec_string = G_PARAM_SPEC_STRING(pspec);
        if (pspec_string->default_value) {
            arg->has_default = 1;
            arg->default_type = 4;  // string
            arg->string_default = strdup(pspec_string->default_value);
        }
    }
    else if (G_IS_PARAM_SPEC_ENUM(pspec)) {
        GParamSpecEnum *pspec_enum = G_PARAM_SPEC_ENUM(pspec);
        arg->has_default = 1;
        arg->default_type = 2;  // int
        arg->int_default = pspec_enum->default_value;
    }
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
    // Get the basic arguments as before
    VipsOperation *op = vips_operation_new(operation_name);
    if (!op) {
        *count = 0;
        return NULL;
    }

    // Get arguments from introspection
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

// Free operation arguments
void free_operation_arguments(ArgInfo *args, int count) {
    if (!args) return;

    for (int i = 0; i < count; i++) {
        free(args[i].name);
        free(args[i].nick);
        free(args[i].blurb);

        // Free string default values if present
        if (args[i].has_default && args[i].default_type == 4 && args[i].string_default) {
            free(args[i].string_default);
        }
    }

    free(args);
}

// Callback to collect operations
static void* collect_operations(GType type, OperationInfo *info, int *count, int max_count) {
    if (!G_TYPE_IS_ABSTRACT(type) && *count < max_count) {
        VipsObjectClass *object_class = VIPS_OBJECT_CLASS(g_type_class_ref(type));

        if (object_class && object_class->nickname) {
            info[*count].name = strdup(object_class->nickname);
            info[*count].nickname = strdup(object_class->nickname);
            info[*count].description = strdup(object_class->description ? object_class->description : "");

            // Create instance to get flags
            VipsOperation *op = VIPS_OPERATION(g_object_new(type, NULL));
            if (op) {
                info[*count].flags = vips_operation_get_flags(op);
                g_object_unref(op);
            } else {
                info[*count].flags = 0;
            }

            (*count)++;
        }

        g_type_class_unref(object_class);
    }

    return NULL;
}

// Function to recursively collect operations
static void collect_operations_recursive(GType type, OperationInfo *info, int *count, int max_count) {
    // Check this type
    collect_operations(type, info, count, max_count);

    // Check child types
    guint n_children = 0;
    GType *children = g_type_children(type, &n_children);

    for (guint i = 0; i < n_children && *count < max_count; i++) {
        collect_operations_recursive(children[i], info, count, max_count);
    }

    g_free(children);
}

// Get all available operations
OperationInfo* get_all_operations(int *count) {
    // Allocate space for a large number of operations
    const int max_ops = 1000;
    OperationInfo *operations = (OperationInfo*)malloc(max_ops * sizeof(OperationInfo));
    *count = 0;

    // Start with the base operation type
    GType base_type = VIPS_TYPE_OPERATION;

    // Recursively collect all operations
    collect_operations_recursive(base_type, operations, count, max_ops);

    return operations;
}

// Free operation information
void free_operation_info(OperationInfo *ops, int count) {
    if (!ops) return;

    for (int i = 0; i < count; i++) {
        free(ops[i].name);
        free(ops[i].nickname);
        free(ops[i].description);
    }

    free(ops);
}

// Get detailed information about an operation
OperationDetails get_operation_details(const char *operation_name) {
    OperationDetails details = {0};

    VipsOperation *op = vips_operation_new(operation_name);
    if (!op) return details;

    int arg_count = 0;
    int image_output_count = 0;
    ArgInfo *args = get_operation_arguments(operation_name, &arg_count);
    if (args) {
        for (int i = 0; i < arg_count; i++) {
            if (args[i].is_input) {
                if (args[i].is_image) {
                    details.has_image_input = 1;
                }
                if (args[i].is_buffer) {
                    details.has_buffer_input = 1;
                }
                GType type = args[i].type_val;
                if (g_type_is_a(type, vips_array_image_get_type())) {
                    details.has_array_image_input = 1;
                }
            }
            if (args[i].is_output) {
                if (args[i].is_image) {
                    details.has_image_output = 1;
                    image_output_count++;
                }
                if (args[i].is_buffer) {
                    details.has_buffer_output = 1;
                }
            }
        }
        details.has_one_image_output = (image_output_count == 1);
        free_operation_arguments(args, arg_count);
    }

    // Get category from filename
    // This is a bit of a hack, but it's how vips itself categorizes operations
    VipsObjectClass *class = VIPS_OBJECT_CLASS(G_OBJECT_GET_CLASS(op));
    if (class && class->nickname) {
        // Try to determine category from operation name patterns
        if (strstr(operation_name, "load") || strstr(operation_name, "save"))
            details.category = strdup("foreign");
        else if (strstr(operation_name, "conv") || strstr(operation_name, "conva"))
            details.category = strdup("convolution");
        else if (strstr(operation_name, "affine") || strstr(operation_name, "resize"))
            details.category = strdup("resample");
        else if (strstr(operation_name, "add") || strstr(operation_name, "subtract"))
            details.category = strdup("arithmetic");
        else
            details.category = strdup("operation");
    }

    g_object_unref(op);
    return details;
}
