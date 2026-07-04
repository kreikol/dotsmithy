// Paquete manifest: parseo y validación del manifiesto dots.yaml, el único
// contrato entre el motor y el contenido.
//
// Responsabilidad: leer dots.yaml, validarlo y exponer un modelo tipado
// (perfiles, capas de stow, paquetes, system, hooks, externals) al resto del
// motor. Es lógica pura y sin efectos: no toca el disco salvo por Load, que solo
// lee. La resolución de capas por perfil (ResolvedLayers) también vive aquí por
// ser una operación puramente derivada del manifiesto.
package manifest
