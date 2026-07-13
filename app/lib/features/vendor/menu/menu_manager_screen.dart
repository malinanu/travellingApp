// Menu manager — port of MenuManager.jsx (icon tiles instead of photos).
// Only reachable for food business types; others get a pointer card.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/theme/palette.dart';
import '../../../core/widgets/press_scale.dart';
import '../../../core/widgets/sheet_scaffold.dart';
import '../../../core/widgets/spotto_icons.dart';
import '../../../data/mock/vendor_seed.dart';
import '../../../data/models/models.dart';
import '../../../data/repositories/repositories.dart';
import '../widgets/vendor_common.dart';

class VendorMenuScreen extends ConsumerWidget {
  const VendorMenuScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final text = Theme.of(context).textTheme;
    final business = ref.watch(vendorAuthProvider);
    final type = businessTypes.where((t) => t.id == business?.typeId).firstOrNull;
    final menu = ref.watch(vendorMenuProvider);

    if (type == null || !type.food) {
      return Scaffold(
        appBar: AppBar(title: const Text('Menu'), backgroundColor: SpottoColors.sand),
        body: Center(
          child: Padding(
            padding: const EdgeInsets.all(32),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                const Icon(Icons.restaurant_menu, size: 40, color: SpottoColors.muted),
                const SizedBox(height: 12),
                Text('Menus are for food businesses',
                    style: text.titleMedium, textAlign: TextAlign.center),
                const SizedBox(height: 6),
                Text(
                    'Switch your business type to Restaurant, Café or Bar in the profile editor to unlock the menu.',
                    style: text.bodyMedium?.copyWith(color: SpottoColors.muted),
                    textAlign: TextAlign.center),
                const SizedBox(height: 16),
                FilledButton(
                  onPressed: () => context.go('/v/more/profile'),
                  child: const Text('Open profile editor'),
                ),
              ],
            ),
          ),
        ),
      );
    }

    return Scaffold(
      appBar: AppBar(title: const Text('Menu'), backgroundColor: SpottoColors.sand),
      floatingActionButton: FloatingActionButton.extended(
        backgroundColor: SpottoColors.accent,
        foregroundColor: Colors.white,
        onPressed: () => _showItemEditor(context, ref),
        icon: const Icon(Icons.add),
        label: const Text('Add item'),
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(20, 8, 20, 90),
        children: [
          for (final s in menu) ...[
            Padding(
              padding: const EdgeInsets.only(left: 4, bottom: 8, top: 8),
              child: Text(s.name.toUpperCase(),
                  style: text.labelSmall
                      ?.copyWith(color: SpottoColors.accent, letterSpacing: 0.6)),
            ),
            for (final item in s.items)
              Padding(
                padding: const EdgeInsets.only(bottom: 10),
                child: VendorCard(
                  padding: const EdgeInsets.all(12),
                  child: Row(
                    children: [
                      Container(
                        width: 44,
                        height: 44,
                        decoration: BoxDecoration(
                          color: SpottoColors.accentTint,
                          borderRadius: BorderRadius.circular(12),
                        ),
                        child: Icon(spottoIcon(item.icon),
                            size: 21, color: SpottoColors.accent),
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(item.name,
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                                style: text.titleSmall),
                            Text(item.desc,
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                                style: text.bodySmall
                                    ?.copyWith(color: SpottoColors.muted)),
                          ],
                        ),
                      ),
                      const SizedBox(width: 8),
                      Text('Rs ${item.price}', style: text.labelLarge),
                      const SizedBox(width: 6),
                      PressScale(
                        scale: 0.9,
                        onTap: () =>
                            _showItemEditor(context, ref, sectionId: s.id, item: item),
                        child: const Padding(
                          padding: EdgeInsets.all(6),
                          child: Icon(Icons.edit_outlined,
                              size: 17, color: SpottoColors.muted),
                        ),
                      ),
                    ],
                  ),
                ),
              ),
          ],
          const SizedBox(height: 12),
          Text('This is exactly what travellers see in your Spotto listing.',
              textAlign: TextAlign.center,
              style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
        ],
      ),
    );
  }

  void _showItemEditor(BuildContext context, WidgetRef ref,
      {String? sectionId, MenuItem? item}) {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      builder: (_) => _MenuItemEditor(sectionId: sectionId, item: item),
    );
  }
}

class _MenuItemEditor extends ConsumerStatefulWidget {
  final String? sectionId;
  final MenuItem? item;
  const _MenuItemEditor({this.sectionId, this.item});

  @override
  ConsumerState<_MenuItemEditor> createState() => _MenuItemEditorState();
}

class _MenuItemEditorState extends ConsumerState<_MenuItemEditor> {
  late final TextEditingController _name =
      TextEditingController(text: widget.item?.name ?? '');
  late final TextEditingController _price =
      TextEditingController(text: widget.item == null ? '' : '${widget.item!.price}');
  late final TextEditingController _desc =
      TextEditingController(text: widget.item?.desc ?? '');
  late String _sectionId =
      widget.sectionId ?? ref.read(vendorMenuProvider).first.id;
  late String _icon = widget.item?.icon ?? 'UtensilsCrossed';

  static const _icons = [
    'EggFried', 'Sandwich', 'Soup', 'Fish', 'Coffee', 'GlassWater',
    'Wine', 'Martini', 'Flame', 'Cookie', 'UtensilsCrossed',
  ];

  @override
  void dispose() {
    _name.dispose();
    _price.dispose();
    _desc.dispose();
    super.dispose();
  }

  void _save() {
    final item = MenuItem(
      id: widget.item?.id ?? 'm${DateTime.now().millisecondsSinceEpoch}',
      name: _name.text.trim().isEmpty ? 'New item' : _name.text.trim(),
      price: int.tryParse(_price.text.trim()) ?? 0,
      desc: _desc.text.trim(),
      icon: _icon,
    );
    ref.read(vendorMenuProvider.notifier).upsertItem(_sectionId, item);
    Navigator.of(context).pop();
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final sections = ref.watch(vendorMenuProvider);

    return SheetScaffold(
      padding:
          EdgeInsets.fromLTRB(20, 4, 20, 24 + MediaQuery.of(context).viewInsets.bottom),
      child: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(widget.item == null ? 'Add menu item' : 'Edit menu item',
                style: text.headlineSmall?.copyWith(fontSize: 19)),
            const SizedBox(height: 16),
            TextField(
                controller: _name,
                decoration: const InputDecoration(hintText: 'Item name')),
            const SizedBox(height: 10),
            TextField(
              controller: _price,
              keyboardType: TextInputType.number,
              decoration: const InputDecoration(hintText: 'Price (Rs)'),
            ),
            const SizedBox(height: 10),
            TextField(
                controller: _desc,
                decoration: const InputDecoration(hintText: 'Short description')),
            const SizedBox(height: 14),
            Text('Section',
                style: text.labelMedium?.copyWith(color: SpottoColors.inkSoft)),
            const SizedBox(height: 8),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: [
                for (final s in sections)
                  PressScale(
                    scale: 0.95,
                    onTap: () => setState(() => _sectionId = s.id),
                    child: Container(
                      padding:
                          const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
                      decoration: BoxDecoration(
                        color:
                            _sectionId == s.id ? SpottoColors.accent : Colors.white,
                        borderRadius: BorderRadius.circular(999),
                        border: Border.all(
                            color: _sectionId == s.id
                                ? SpottoColors.accent
                                : SpottoColors.line,
                            width: 1.5),
                      ),
                      child: Text(s.name,
                          style: TextStyle(
                              fontSize: 13,
                              fontWeight: FontWeight.w800,
                              color: _sectionId == s.id
                                  ? Colors.white
                                  : SpottoColors.ink)),
                    ),
                  ),
              ],
            ),
            const SizedBox(height: 14),
            Text('Icon', style: text.labelMedium?.copyWith(color: SpottoColors.inkSoft)),
            const SizedBox(height: 8),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: [
                for (final icon in _icons)
                  PressScale(
                    scale: 0.92,
                    onTap: () => setState(() => _icon = icon),
                    child: Container(
                      width: 42,
                      height: 42,
                      decoration: BoxDecoration(
                        color: _icon == icon
                            ? SpottoColors.accent
                            : SpottoColors.accentTint,
                        borderRadius: BorderRadius.circular(12),
                      ),
                      child: Icon(spottoIcon(icon),
                          size: 19,
                          color: _icon == icon ? Colors.white : SpottoColors.accent),
                    ),
                  ),
              ],
            ),
            const SizedBox(height: 18),
            Row(
              children: [
                if (widget.item != null) ...[
                  Expanded(
                    child: OutlinedButton(
                      style: OutlinedButton.styleFrom(foregroundColor: vendorRed),
                      onPressed: () {
                        ref
                            .read(vendorMenuProvider.notifier)
                            .removeItem(widget.item!.id);
                        Navigator.of(context).pop();
                      },
                      child: const Text('Delete'),
                    ),
                  ),
                  const SizedBox(width: 10),
                ],
                Expanded(
                  flex: 2,
                  child: FilledButton(
                    onPressed: _save,
                    child: Text(widget.item == null ? 'Add to menu' : 'Save changes'),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
