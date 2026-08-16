import 'package:flutter/material.dart';
import '../../../core/theme/palette.dart';
import '../../../core/widgets/spotto_icons.dart';
import '../../../core/widgets/press_scale.dart';
import '../../../data/vendor_categories.dart';

Future<Set<String>?> showCategoriesSheet(
  BuildContext context, {
  required Set<String> selectedCategories,
  required List<VendorCategory> availableCategories,
}) {
  return showModalBottomSheet<Set<String>>(
    context: context,
    isScrollControlled: true,
    backgroundColor: const Color(0xFF141414),
    useSafeArea: true,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
    ),
    builder: (context) {
      return _CategoriesSheet(
        initialSelection: selectedCategories,
        availableCategories: availableCategories,
      );
    },
  );
}

class _CategoriesSheet extends StatefulWidget {
  final Set<String> initialSelection;
  final List<VendorCategory> availableCategories;

  const _CategoriesSheet({
    required this.initialSelection,
    required this.availableCategories,
  });

  @override
  State<_CategoriesSheet> createState() => _CategoriesSheetState();
}

class _CategoriesSheetState extends State<_CategoriesSheet> {
  late Set<String> _selected;
  final ScrollController _scrollController = ScrollController();

  @override
  void initState() {
    super.initState();
    _selected = Set.from(widget.initialSelection);
  }

  void _toggleCategory(String label) {
    setState(() {
      if (label == 'All') {
        _selected.clear();
        _selected.add('All');
        return;
      }
      
      _selected.remove('All');
      
      if (_selected.contains(label)) {
        _selected.remove(label);
        if (_selected.isEmpty) {
          _selected.add('All');
        }
      } else {
        _selected.add(label);
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final grouped = <VendorSegment, List<VendorCategory>>{};
    for (final seg in vendorSegments) {
      grouped[seg.id] = [];
    }
    for (final cat in widget.availableCategories) {
      if (grouped.containsKey(cat.segment)) {
        grouped[cat.segment]!.add(cat);
      }
    }
    grouped.removeWhere((key, value) => value.isEmpty);

    return Scaffold(
      backgroundColor: Colors.transparent,
      body: Column(
        children: [
          // Header
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 24, 16, 16),
            child: Row(
              children: [
                GestureDetector(
                  onTap: () => Navigator.pop(context, _selected),
                  child: const Icon(Icons.arrow_back_rounded, color: Colors.white, size: 24),
                ),
                const Expanded(
                  child: Center(
                    child: Text(
                      'More',
                      style: TextStyle(
                        fontSize: 18,
                        fontWeight: FontWeight.bold,
                        color: Colors.white,
                      ),
                    ),
                  ),
                ),
                GestureDetector(
                  onTap: () {
                    setState(() {
                      _selected.clear();
                      _selected.add('All');
                    });
                  },
                  child: const Text('Clear', style: TextStyle(color: Colors.transparent)), // Hidden for symmetry, or you can keep it white
                ),
              ],
            ),
          ),
          
          // Segments Jump Row
          SizedBox(
            height: 48,
            child: ListView(
              scrollDirection: Axis.horizontal,
              padding: const EdgeInsets.symmetric(horizontal: 16),
              children: grouped.keys.map((segment) {
                final label = vendorSegments.firstWhere((s) => s.id == segment).label;
                final isSelectedSegment = grouped.keys.first == segment; // Simplistic active tab
                return Padding(
                  padding: const EdgeInsets.only(right: 8),
                  child: Center(
                    child: Container(
                      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
                      decoration: BoxDecoration(
                        color: isSelectedSegment ? const Color(0xFF333333) : Colors.transparent,
                        borderRadius: BorderRadius.circular(8),
                      ),
                      child: Text(
                        label,
                        style: TextStyle(
                          fontSize: 14,
                          fontWeight: FontWeight.w600,
                          color: isSelectedSegment ? Colors.white : SpottoColors.muted,
                        ),
                      ),
                    ),
                  ),
                );
              }).toList(),
            ),
          ),
          const Divider(height: 1, color: Color(0xFF2C2C2C)),

          // Scrollable Groups
          Expanded(
            child: SingleChildScrollView(
              controller: _scrollController,
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 24),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  for (final segment in grouped.keys)
                    _buildSegmentGroup(segment, grouped[segment]!),
                ],
              ),
            ),
          ),
          
          // Apply Button
          SafeArea(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: SizedBox(
                width: double.infinity,
                height: 54,
                child: FilledButton(
                  style: FilledButton.styleFrom(
                    backgroundColor: SpottoColors.accent,
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(16),
                    ),
                  ),
                  onPressed: () => Navigator.pop(context, _selected),
                  child: Text(
                    'Apply (${_selected.contains('All') ? 'All' : _selected.length})',
                    style: const TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
                  ),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildSegmentGroup(VendorSegment segment, List<VendorCategory> categories) {
    final label = vendorSegments.firstWhere((s) => s.id == segment).label;
    
    return Padding(
      padding: const EdgeInsets.only(bottom: 32),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            label,
            style: const TextStyle(
              fontSize: 18,
              fontWeight: FontWeight.bold,
              color: Colors.white,
            ),
          ),
          const SizedBox(height: 16),
          GridView.builder(
            shrinkWrap: true,
            physics: const NeverScrollableScrollPhysics(),
            padding: EdgeInsets.zero,
            gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
              crossAxisCount: 2,
              mainAxisSpacing: 12,
              crossAxisSpacing: 12,
              childAspectRatio: 3.5, // Matches the elongated chip shape
            ),
            itemCount: categories.length,
            itemBuilder: (context, index) {
              final cat = categories[index];
              final isSelected = _selected.contains(cat.label);
              // Use a dark teal matching the screenshot
              final bgColor = isSelected ? SpottoColors.deepTeal : const Color(0xFF04343F);
              
              return PressScale(
                scale: 0.95,
                onTap: () => _toggleCategory(cat.label),
                child: Container(
                  padding: const EdgeInsets.symmetric(horizontal: 12),
                  alignment: Alignment.center,
                  decoration: BoxDecoration(
                    color: bgColor,
                    borderRadius: BorderRadius.circular(999),
                    border: Border.all(
                      color: isSelected ? Colors.white24 : Colors.transparent,
                      width: 1,
                    ),
                  ),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      Icon(
                        spottoIcon(cat.icon),
                        size: 16,
                        color: Colors.white, // light cyan/white
                      ),
                      const SizedBox(width: 8),
                      Flexible(
                        child: Text(
                          cat.label,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: const TextStyle(
                            fontSize: 14,
                            fontWeight: FontWeight.w500,
                            color: Colors.white,
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
              );
            },
          ),
        ],
      ),
    );
  }
}
